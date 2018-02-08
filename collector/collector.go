package collector

import (
	"bytes"
	"encoding/csv"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

type metric struct {
	metricsname string
	value       float64
	unit        string
}

// Exporter implements the prometheus.Collector interface. It exposes the metrics
// of a ipmi node.
type Exporter struct {
	IPMIBinary string

	namespace string
}

// NewExporter instantiates a new ipmi Exporter.
func NewExporter(ipmiBinary string) *Exporter {
	return &Exporter{
		IPMIBinary: ipmiBinary,
		namespace:  "ipmi",
	}
}

func ipmiOutput(cmd string) ([]byte, error) {
	parts := strings.Fields(cmd)
	out, err := exec.Command(parts[0], parts[1:]...).Output()
	if err != nil {
		log.Errorf("error while calling ipmitool: %v", err)
	}
	return out, err
}

func convertValue(strfloat string) (value float64, err error) {
	if strfloat != "N/A" {
		value, err = strconv.ParseFloat(strfloat, 64)
	}
	return value, err
}

func convertOutput(result [][]string) (metrics []metric, err error) {
	for _, res := range result {
		var value float64
		var currentMetric metric

		for n := range res {
			res[n] = strings.TrimSpace(res[n])
		}
		value, err = convertValue(res[3])
		if err != nil {
			log.Errorf("could not parse ipmi output: %s", err)
		}

		currentMetric.value = value
		currentMetric.unit = res[4]
		currentMetric.metricsname = res[1]

		metrics = append(metrics, currentMetric)
	}
	return metrics, err
}

func splitOutput(impiOutput []byte) ([][]string, error) {
	r := csv.NewReader(bytes.NewReader(impiOutput))
	r.Comma = '|'
	r.Comment = '#'
	result, err := r.ReadAll()
	if err != nil {
		log.Errorf("could not parse ipmi output: %v", err)
		return result, err
	}

	keys := make(map[string]int)
	var res [][]string
	for _, v := range result {
		key := v[1]
		if _, ok := keys[key]; ok {
			keys[key] += 1
			v[1] = strings.TrimSpace(v[1]) + strconv.Itoa(keys[key])
		} else {
			keys[key] = 1
		}
		res = append(res, v)
	}
	return res, err
}

// Describe describes all the registered stats metrics from the ipmi node.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- temperatures
	ch <- fanspeed
	ch <- voltages
	ch <- intrusion
	ch <- powersupply
	ch <- current
}

// Collect collects all the registered stats metrics from the ipmi node.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	output, err := ipmiOutput("ipmi-sensors -Q --sdr-cache-recreate --no-header-output")
	if err != nil {
		log.Errorln(err)
	}
	splitted, err := splitOutput(output)
	if err != nil {
		log.Errorln(err)
	}
	convertedOutput, err := convertOutput(splitted)
	if err != nil {
		log.Errorln(err)
	}

	for _, res := range convertedOutput {
		push := func(m *prometheus.Desc) {
			ch <- prometheus.MustNewConstMetric(m, prometheus.GaugeValue, res.value, res.metricsname)
		}

		switch strings.ToLower(res.unit) {
		case "c":
			push(temperatures)
		case "v":
			push(voltages)
		case "rpm":
			push(fanspeed)
		case "w":
			push(powersupply)
		case "a":
			push(current)
		}
	}
}
