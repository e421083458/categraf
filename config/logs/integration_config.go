// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package logs

import (
	"fmt"
	"strings"
)

// Logs source types
const (
	TCPType           = "tcp"
	UDPType           = "udp"
	FileType          = "file"
	DockerType        = "docker"
	JournaldType      = "journald"
	WindowsEventType  = "windows_event"
	SnmpTrapsType     = "snmp_traps"
	StringChannelType = "string_channel"

	// UTF16BE for UTF-16 Big endian encoding
	UTF16BE string = "utf-16-be"
	// UTF16LE for UTF-16 Little Endian encoding
	UTF16LE string = "utf-16-le"

	// https://en.wikipedia.org/wiki/GB_2312
	// https://en.wikipedia.org/wiki/GBK_(character_encoding)
	// https://en.wikipedia.org/wiki/GB_18030
	// https://en.wikipedia.org/wiki/Big5
	GB18030  string = "gb18030"
	GB2312   string = "gb2312"
	HZGB2312 string = "hz-gb2312"
	GBK      string = "gbk"
	BIG5     string = "big5"
)

// LogsConfig represents a log source config, which can be for instance
// a file to tail or a port to listen to.
type (
	LogsConfig struct {
		Type string

		Port        int    // Network
		IdleTimeout string `mapstructure:"idle_timeout" json:"idle_timeout" toml:"idle_timeout"` // Network
		Path        string // File, Journald

		Encoding     string   `mapstructure:"encoding" json:"encoding" toml:"encoding"`                   // File
		ExcludePaths []string `mapstructure:"exclude_paths" json:"exclude_paths" toml:"exclude_paths"`    // File
		TailingMode  string   `mapstructure:"start_position" json:"start_position" toml:"start_position"` // File

		IncludeUnits  []string `mapstructure:"include_units" json:"include_units" toml:"include_units"`    // Journald
		ExcludeUnits  []string `mapstructure:"exclude_units" json:"exclude_units" toml:"exclude_units"`    // Journald
		ContainerMode bool     `mapstructure:"container_mode" json:"container_mode" toml:"container_mode"` // Journald

		Image string // Docker
		Label string // Docker
		// Name contains the container name
		Name string // Docker
		// Identifier contains the container ID
		Identifier string // Docker

		ChannelPath string `mapstructure:"channel_path" json:"channel_path" toml:"channel_path"` // Windows Event
		Query       string // Windows Event

		// used as input only by the Channel tailer.
		// could have been unidirectional but the tailer could not close it in this case.
		Channel chan *ChannelMessage `json:"-"`

		Service         string
		Source          string
		SourceCategory  string
		Tags            []string
		ProcessingRules []*ProcessingRule `mapstructure:"log_processing_rules" json:"log_processing_rules" toml:"log_processing_rules"`

		AutoMultiLine               bool    `mapstructure:"auto_multi_line_detection" json:"auto_multi_line_detection" toml:"auto_multi_line_detectio"`
		AutoMultiLineSampleSize     int     `mapstructure:"auto_multi_line_sample_size" json:"auto_multi_line_sample_size" toml:"auto_multi_line_sample_size"`
		AutoMultiLineMatchThreshold float64 `mapstructure:"auto_multi_line_match_threshold" json:"auto_multi_line_match_threshold" toml:"auto_multi_line_match_threshold"`
	}
)

// TailingMode type
type TailingMode uint8

// Tailing Modes
const (
	ForceBeginning = iota
	ForceEnd
	Beginning
	End
)

var tailingModeTuples = []struct {
	s string
	m TailingMode
}{
	{"forceBeginning", ForceBeginning},
	{"forceEnd", ForceEnd},
	{"beginning", Beginning},
	{"end", End},
}

// TailingModeFromString parses a string and returns a corresponding tailing mode, default to End if not found
func TailingModeFromString(mode string) (TailingMode, bool) {
	for _, t := range tailingModeTuples {
		if t.s == mode {
			return t.m, true
		}
	}
	return End, false
}

// TailingModeToString returns seelog string representation for a specified tailing mode. Returns "" for invalid tailing mode.
func (mode TailingMode) String() string {
	for _, t := range tailingModeTuples {
		if t.m == mode {
			return t.s
		}
	}
	return ""
}

// Validate returns an error if the config is misconfigured
func (c *LogsConfig) Validate() error {
	switch {
	case c.Type == "":
		// user don't have to specify a logs-config type when defining
		// an autodiscovery label because so we must override it at some point,
		// this check is mostly used for sanity purposed to detect an override miss.
		return fmt.Errorf("a config must have a type")
	case c.Type == FileType:
		if c.Path == "" {
			return fmt.Errorf("file source must have a path")
		}
		err := c.validateTailingMode()
		if err != nil {
			return err
		}
	case c.Type == TCPType && c.Port == 0:
		return fmt.Errorf("tcp source must have a port")
	case c.Type == UDPType && c.Port == 0:
		return fmt.Errorf("udp source must have a port")
	}
	err := ValidateProcessingRules(c.ProcessingRules)
	if err != nil {
		return err
	}
	return CompileProcessingRules(c.ProcessingRules)
}

func (c *LogsConfig) validateTailingMode() error {
	mode, found := TailingModeFromString(c.TailingMode)
	if !found && c.TailingMode != "" {
		return fmt.Errorf("invalid tailing mode '%v' for %v", c.TailingMode, c.Path)
	}
	if ContainsWildcard(c.Path) && (mode == Beginning || mode == ForceBeginning) {
		return fmt.Errorf("tailing from the beginning is not supported for wildcard path %v", c.Path)
	}
	return nil
}

// ContainsWildcard returns true if the path contains any wildcard character
func ContainsWildcard(path string) bool {
	return strings.ContainsAny(path, "*?[")
}
