package inventory

type Kind string

const (
	KindCheckpoints     Kind = "checkpoints"
	KindLoras           Kind = "loras"
	KindTextEncoders    Kind = "text_encoders"
	KindControlNet      Kind = "controlnet"
	KindVAE             Kind = "vae"
	KindConfigs         Kind = "configs"
	KindDiffusionModels Kind = "diffusion_models"
	KindHypernetworks   Kind = "hypernetworks"
	KindStyleModels     Kind = "style_models"
	KindClipVision      Kind = "clip_vision"
	KindAudioEncoders   Kind = "audio_encoders"
)

func AllKinds() []Kind {
	return []Kind{
		KindAudioEncoders,
		KindCheckpoints,
		KindClipVision,
		KindConfigs,
		KindControlNet,
		KindDiffusionModels,
		KindHypernetworks,
		KindLoras,
		KindStyleModels,
		KindTextEncoders,
		KindVAE,
	}
}

func ParseKind(raw string) (Kind, bool) {
	for _, kind := range AllKinds() {
		if kind == Kind(raw) {
			return Kind(raw), true
		}
	}
	return "", false
}
