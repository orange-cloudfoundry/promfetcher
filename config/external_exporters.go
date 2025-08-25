package config

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	txttpl "text/template"

	"github.com/orange-cloudfoundry/promfetcher/models"
)

type ExternalExporters []*ExternalExporter

type ExternalExporter struct {
	Name        string                     `yaml:"name"`
	Host        string                     `yaml:"host"`
	MetricsPath string                     `yaml:"metrics_path"`
	Scheme      string                     `yaml:"scheme"`
	Params      map[string][]ValueTemplate `yaml:"params"`
	IsTls       bool                       `yaml:"-"`
}

func (ee *ExternalExporter) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain ExternalExporter
	err := unmarshal((*plain)(ee))
	if err != nil {
		return err
	}
	if ee.Host == "" {
		return fmt.Errorf("host must be provided on external exporter")
	}
	if ee.MetricsPath == "" {
		ee.MetricsPath = "/metrics"
	}
	if ee.Name == "" {
		ee.Name = ee.Host + ee.MetricsPath
	}
	if ee.Scheme == "" {
		ee.Scheme = "http"
	}
	if ee.Scheme == "https" {
		ee.IsTls = true
	}

	return nil
}

func (ee *ExternalExporter) ToRoute(tags models.Tags) (*models.Route, error) {
	urlValues, err := ee.ParamsToURLValues(tags)
	if err != nil {
		return nil, fmt.Errorf("error on external exporter `%s`: %s", ee.Name, err.Error())
	}
	return &models.Route{
		PrivateInstanceID: ee.Name,
		Tags:              tags,
		Address:           ee.Host,
		TLS:               ee.IsTls,
		URLParams:         urlValues,
		MetricsPath:       ee.MetricsPath,
		Host:              ee.Host,
	}, nil
}

func (ee *ExternalExporter) ParamsToURLValues(tags models.Tags) (url.Values, error) {
	urlValue := make(url.Values)
	var err error
	for key, values := range ee.Params {
		finalValues := make([]string, len(values))
		for i, valueTpl := range values {
			finalValues[i], err = valueTpl.ResolveTags(tags)
			if err != nil {
				return nil, err
			}
		}
		urlValue[key] = finalValues
	}
	return urlValue, nil
}

type ValueTemplate struct {
	Raw string
	tpl *txttpl.Template
}

func (vt *ValueTemplate) UnmarshalYAML(unmarshal func(interface{}) error) error {
	rawString := ""
	err := unmarshal(&rawString)
	if err != nil {
		return err
	}
	vt.Raw = rawString
	if !strings.Contains(vt.Raw, "{{") {
		return nil
	}

	vt.tpl, err = txttpl.New("").Parse(vt.Raw)
	if err != nil {
		return err
	}
	return nil
}

func (vt *ValueTemplate) ResolveTags(tags models.Tags) (string, error) {
	if vt.tpl == nil {
		return vt.Raw, nil
	}
	buf := &bytes.Buffer{}
	err := vt.tpl.Execute(buf, tags)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
