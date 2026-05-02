package config

func ApplyDefaults(cfg *Config) {
	if cfg == nil {
		return
	}

	if len(cfg.Ignore.Paths) == 0 {
		cfg.Ignore.Paths = DefaultIgnorePaths()
	}

	if len(cfg.CPP.IncludePaths) == 0 {
		cfg.CPP.IncludePaths = []string{"src", "include"}
	}

	if !cfg.CPP.CompileCommands.AutoDiscover && cfg.CPP.CompileCommands.Path == "" {
		cfg.CPP.CompileCommands.AutoDiscover = true
	}
}

func DefaultIgnorePaths() []string {
	return []string{
		".git/**",
		"build/**",
		"cmake-build-debug/**",
		"cmake-build-release/**",
		"node_modules/**",
		"vendor/**",
		"libs/**",
		"third_party/**",
		"external/**",
		"generated/**",
		"**/*.pb.h",
		"**/*.pb.cc",
		"**/*.pb.cpp",
		"**/*.pb.go",
		"**/*.grpc.pb.h",
		"**/*.grpc.pb.cc",
		"**/*.grpc.pb.go",
	}
}
