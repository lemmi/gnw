package main

import (
	"fmt"
	"os"

	"github.com/lemmi/closer"
)

type loadavg struct {
	load1    float64
	load5    float64
	load15   float64
	runnable int
	procs    int
	recent   int
}

func readLoadavg() (loadavg, error) {
	var load loadavg
	fLoad, err := os.Open("/proc/loadavg")
	if err != nil {
		return load, err
	}
	defer closer.Do(fLoad)

	_, err = fmt.Fscanf(fLoad, "%f %f %f %d/%d %d",
		&load.load1,
		&load.load5,
		&load.load15,
		&load.runnable,
		&load.procs,
		&load.recent,
	)

	return load, err
}
