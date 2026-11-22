package task

import (
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type state int

const (
	Pending state = iota
	Scheduled
	Running
	Completed
	Failed
)

type Task struct {
	ID            uuid.UUID
	Name          string
	State         state
	Image         string
	Memory        int
	Disk          int
	ExposedPorts  nat.PortSet
	PortBinding   map[string]string
	RestartPolicy string
	startTime     time.Time
	endTime       time.Time
}

type TaskEvent struct {
	ID        uuid.UUID
	State     state
	Timestamp time.Time
	Task      Task
}
