package cli

import (
	"fmt"

	"github.com/DishanRajapaksha/iec-104-cli/internal/config"
	"github.com/DishanRajapaksha/iec-104-cli/internal/iec104"
)

type pointFilter func(iec104.PointValue) (iec104.PointValue, bool)

func buildPointFilter(cfg config.Config, commonAddress uint16, ioa uint32, pointName string) (pointFilter, error) {
	pointsByName := map[string]config.PointConfig{}
	pointsByIOA := map[uint32]config.PointConfig{}
	for _, point := range cfg.Points {
		pointsByName[point.Name] = point
		pointsByIOA[point.IOA] = point
	}
	if pointName != "" {
		point, ok := pointsByName[pointName]
		if !ok {
			return nil, fmt.Errorf("unknown point %q", pointName)
		}
		ioa = point.IOA
	}

	return func(value iec104.PointValue) (iec104.PointValue, bool) {
		if commonAddress != 0 && value.CommonAddress != commonAddress {
			return value, false
		}
		if ioa != 0 && value.IOA != ioa {
			return value, false
		}
		if point, ok := pointsByIOA[value.IOA]; ok {
			value.Name = point.Name
			value.Unit = point.Unit
			if value.Type == "" {
				value.Type = point.Type
			}
		}
		return value, true
	}, nil
}
