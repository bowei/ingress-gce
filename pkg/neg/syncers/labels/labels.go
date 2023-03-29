package labels

// PodLabelPropagationConfig contains a list of configurations for labels to be propagated to GCE network endpoints.
type PodLabelPropagationConfig struct {
	Labels []Label
}

// Label contains configuration for a label to be propagated to GCE network endpoints.
type Label struct {
	Key               string
	ShortKey          string
	MaxLabelSizeBytes int
}
