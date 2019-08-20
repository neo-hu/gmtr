package gmtr

import (
	"encoding/json"
	"strings"
)

type StringSet map[string]struct{}

func (ms StringSet) MarshalJSON() ([]byte, error) {
	val := make([]string, 0, len(ms))
	for k := range ms {
		val = append(val, k)
	}
	return json.Marshal(val)
}

func (ms *StringSet) UnmarshalJSON(b []byte) error {
	val := []string{}
	err := json.Unmarshal(b, &val)
	if err != nil {
		return err
	}
	*ms = make(StringSet)
	for _, v := range val {
		ms.Add(v)
	}
	return nil
}

func (ms StringSet) Values() []string {
	var result []string
	for key, _ := range ms {
		result = append(result, key)
	}
	return result
}
func (ms StringSet) String() string {
	var values []string
	for metric, _ := range ms {
		values = append(values, string(metric))
	}
	return strings.Join(values, ",")
}

func (ms StringSet) Has(mk string) bool {
	_, exists := ms[mk]
	return exists
}

func (ms StringSet) Add(mk string) {
	ms[mk] = struct{}{}
}

func (ms StringSet) Copy() StringSet {
	d := StringSet{}
	for key, _ := range ms {
		d.Add(key)
	}
	return d
}
