package finance

import (
	"strings"

	exchangerate "codeberg.org/urutau-ltd/gavia/internal/models/exchange_rate"
)

type Converter struct {
	graph map[string]map[string]float64
}

func NewConverter(samples []*exchangerate.Sample) *Converter {
	graph := make(map[string]map[string]float64)
	for _, sample := range samples {
		if sample == nil || sample.Rate <= 0 {
			continue
		}

		base := strings.ToUpper(strings.TrimSpace(sample.BaseCurrency))
		quote := strings.ToUpper(strings.TrimSpace(sample.QuoteCurrency))
		if base == "" || quote == "" || base == quote {
			continue
		}

		if graph[base] == nil {
			graph[base] = make(map[string]float64)
		}
		if graph[quote] == nil {
			graph[quote] = make(map[string]float64)
		}

		graph[base][quote] = sample.Rate
		graph[quote][base] = 1 / sample.Rate
	}

	return &Converter{graph: graph}
}

func (c *Converter) Convert(amount float64, fromCurrency, toCurrency string) (float64, bool) {
	fromCurrency = strings.ToUpper(strings.TrimSpace(fromCurrency))
	toCurrency = strings.ToUpper(strings.TrimSpace(toCurrency))
	if fromCurrency == "" || toCurrency == "" {
		return 0, false
	}
	if fromCurrency == toCurrency {
		return amount, true
	}

	rate, ok := c.rate(fromCurrency, toCurrency, map[string]bool{})
	if !ok {
		return 0, false
	}

	return amount * rate, true
}

func (c *Converter) rate(fromCurrency, toCurrency string, visited map[string]bool) (float64, bool) {
	if c == nil || c.graph == nil {
		return 0, false
	}

	if rate, ok := c.graph[fromCurrency][toCurrency]; ok {
		return rate, true
	}

	visited[fromCurrency] = true
	for nextCurrency, nextRate := range c.graph[fromCurrency] {
		if visited[nextCurrency] {
			continue
		}

		remainingRate, ok := c.rate(nextCurrency, toCurrency, visited)
		if ok {
			return nextRate * remainingRate, true
		}
	}

	return 0, false
}
