package validation

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	telemetryv1alpha1 "github.com/kyma-project/kyma/components/telemetry-operator/api/v1alpha1"
)

//go:generate mockery --name PluginValidator --filename plugin_validator.go
type PluginValidator interface {
	Validate(logPipeline *telemetryv1alpha1.LogPipeline, logPipelines *telemetryv1alpha1.LogPipelineList) error
	ContainsCustomPlugin(logPipeline *telemetryv1alpha1.LogPipeline) bool
}

type pluginValidator struct {
	deniedFilterPlugins []string
	deniedOutputPlugins []string
}

func NewPluginValidator(deniedFilterPlugins, deniedOutputPlugins []string) PluginValidator {
	return &pluginValidator{
		deniedFilterPlugins: deniedFilterPlugins,
		deniedOutputPlugins: deniedOutputPlugins,
	}
}

// ContainsCustomPlugin returns true if the pipeline
// contains any custom filters or outputs
func (pv *pluginValidator) ContainsCustomPlugin(logPipeline *telemetryv1alpha1.LogPipeline) bool {
	for _, filterPlugin := range logPipeline.Spec.Filters {
		if filterPlugin.Custom != "" {
			return true
		}
	}
	if logPipeline.Spec.Output.Custom != "" {
		return true
	}
	for _, f := range logPipeline.Spec.Filters {
		if f.Custom != "" {
			return true
		}
	}
	return false
}

// Validate returns an error if validation fails
// because of using denied plugins or errors in match conditions
// for filters or outputs.
func (pv *pluginValidator) Validate(logPipeline *telemetryv1alpha1.LogPipeline, logPipelines *telemetryv1alpha1.LogPipelineList) error {
	err := pv.validateFilters(logPipeline, logPipelines)
	if err != nil {
		return errors.Wrap(err, "error validating filter plugins")
	}
	err = pv.validateOutput(logPipeline, logPipelines)
	if err != nil {
		return errors.Wrap(err, "error validating output plugin")
	}
	return nil
}

func (pv *pluginValidator) validateFilters(pipeline *telemetryv1alpha1.LogPipeline, pipelines *telemetryv1alpha1.LogPipelineList) error {
	for _, filterPlugin := range pipeline.Spec.Filters {
		if err := checkIfPluginIsValid(filterPlugin.Custom, pipeline, pv.deniedFilterPlugins, pipelines); err != nil {
			return err
		}
	}
	return nil
}

func (pv *pluginValidator) validateOutput(pipeline *telemetryv1alpha1.LogPipeline, pipelines *telemetryv1alpha1.LogPipelineList) error {
	if len(pipeline.Spec.Output.Custom) == 0 {
		return fmt.Errorf("no output is defined, you must define one output")
	}
	if err := checkIfPluginIsValid(pipeline.Spec.Output.Custom, pipeline, pv.deniedOutputPlugins, pipelines); err != nil {
		return err
	}
	return nil
}

func checkIfPluginIsValid(content string, pipeline *telemetryv1alpha1.LogPipeline, denied []string, pipelines *telemetryv1alpha1.LogPipelineList) error {
	customSection, err := parseSection(content)
	if err != nil {
		return err
	}
	name, err := getCustomName(customSection)
	if err != nil {
		return err
	}

	for _, deniedPlugin := range denied {
		if strings.EqualFold(name, deniedPlugin) {
			return fmt.Errorf("plugin '%s' is not supported. ", name)
		}
	}

	if hasMatchCondition(customSection) {
		return fmt.Errorf("plugin '%s' contains match condition. Match conditions are forbidden", name)
	}

	return nil
}

func getCustomName(custom map[string]string) (string, error) {
	if name, hasKey := custom["name"]; hasKey {
		return name, nil
	}
	return "", fmt.Errorf("configuration section does not have name attribute")
}

func hasMatchCondition(section map[string]string) bool {
	if _, hasKey := section["match"]; hasKey {
		return true
	}
	return false
}

func parseSection(section string) (map[string]string, error) {
	result := make(map[string]string)

	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, " ")
		if !found {
			return nil, fmt.Errorf("invalid line: %s", line)
		}
		result[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}
	return result, nil
}