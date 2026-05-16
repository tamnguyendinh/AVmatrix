package embeddings

func NormalizeConfig(config Config) Config {
	defaults := DefaultConfig()
	if config.ModelID == "" {
		config.ModelID = defaults.ModelID
	}
	if config.BatchSize <= 0 {
		config.BatchSize = defaults.BatchSize
	}
	if config.Dimensions <= 0 {
		config.Dimensions = defaults.Dimensions
	}
	if config.Device == "" {
		config.Device = defaults.Device
	}
	if config.MaxSnippetLength <= 0 {
		config.MaxSnippetLength = defaults.MaxSnippetLength
	}
	if config.ChunkSize <= 0 {
		config.ChunkSize = defaults.ChunkSize
	}
	if config.Overlap <= 0 {
		config.Overlap = defaults.Overlap
	}
	if config.MaxDescriptionLength <= 0 {
		config.MaxDescriptionLength = defaults.MaxDescriptionLength
	}
	return config
}
