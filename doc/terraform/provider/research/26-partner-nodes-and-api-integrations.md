# 26 — Partner Nodes and API Integrations

## Overview

ComfyUI includes **180 "Partner Nodes"** (also called API nodes) that integrate with
third-party AI services. These live in the `comfy_api_nodes/` directory of the ComfyUI
source tree and appear in the ComfyUI UI under **"Partner Nodes"** organized into five
subcategories: Image, Video, Audio, Text, and 3D.

### How Partner Nodes Differ from Core Nodes

| Aspect | Core Nodes | Partner Nodes (API Nodes) |
|--------|-----------|--------------------------|
| Execution | Run locally on the ComfyUI server | Call external third-party APIs |
| GPU required | Yes (most) | No — computation happens remotely |
| Authentication | None (local) | Requires API keys for each service |
| Latency | Depends on local hardware | Depends on network + service queue |
| Cost | Free (hardware costs) | Per-request billing by the API provider |
| Source location | `nodes.py`, `comfy_extras/` | `comfy_api_nodes/` |
| Category prefix | Various (`sampling`, `conditioning`, etc.) | `api node/<type>/<provider>` |

Partner nodes are first-class ComfyUI nodes — they appear in workflows alongside core
nodes and can be connected to any compatible input/output. The key difference is that
when a workflow executes, partner nodes send requests to external APIs instead of
running inference locally.

## Architecture

```
┌─────────────────────────────────────────────────┐
│  ComfyUI Workflow (Terraform: comfyui_workflow)  │
│                                                  │
│  ┌──────────┐   ┌──────────────┐   ┌─────────┐ │
│  │ Core Node│──▶│ Partner Node │──▶│Core Node│ │
│  │ (local)  │   │ (API call)   │   │ (local) │ │
│  └──────────┘   └──────┬───────┘   └─────────┘ │
│                         │                        │
└─────────────────────────┼────────────────────────┘
                          │ HTTPS
                          ▼
              ┌───────────────────────┐
              │  Third-Party API      │
              │  (Kling, OpenAI,      │
              │   Recraft, etc.)      │
              └───────────────────────┘
```

### Key Architectural Points

1. **External API calls** — Partner nodes do NOT use the local ComfyUI server for
   inference. They make HTTPS requests to third-party services (Kling AI, OpenAI,
   Stability AI, Recraft, etc.).

2. **API key management** — Each service requires its own API key, typically
   configured in ComfyUI's settings or passed as node inputs. In Terraform, these
   are provider-level or resource-level attributes.

3. **Async execution** — Many partner nodes (especially video generation) use
   polling: submit a job, then poll for completion. ComfyUI handles this
   transparently.

4. **Result caching** — Generated outputs (images, videos, audio) are downloaded
   and stored locally, then passed through ComfyUI's normal output pipeline.

## Categories

### Image (61 nodes, 19 providers)

Image partner nodes handle generation, editing, upscaling, style transfer, and
background removal via external APIs.

| Provider | Count | Capabilities |
|----------|-------|-------------|
| **Recraft** | 15 | Text-to-image, image-to-image, inpainting, vectorization, upscale, background removal/replacement, style library |
| **BFL (Flux)** | 5 | Flux Pro/Ultra generation, Kontext Pro, image expand, image fill |
| **Stability AI** | 5 | Stable Image SD 3.5, Ultra, upscale (conservative/creative/fast) |
| **Magnific** | 5 | Upscale (creative/precise), relight, skin enhancer, style transfer |
| **Gemini** | 3 | Image generation (multiple models), Nano Banana |
| **Ideogram** | 3 | Text-to-image V1, V2, V3 |
| **Kling** | 3 | Image generation, OmniPro image, virtual try-on |
| **Luma** | 3 | Image generation, image modify, reference-based generation |
| **OpenAI** | 3 | DALL-E 2, DALL-E 3, GPT Image 1 |
| **Reve** | 3 | Image create, edit, remix |
| **Bria** | 2 | Image editing, background removal |
| **ByteDance** | 2 | Image generation, Seedream |
| **Grok** | 2 | Image generation, image editing |
| **Quiver** | 2 | Image-to-SVG, text-to-SVG |
| **Wan** | 2 | Image-to-image, text-to-image |
| **HitPaw** | 1 | General image enhancement |
| **Runway** | 1 | Text-to-image |
| **Topaz** | 1 | Image enhancement |
| **WaveSpeed** | 1 | Image upscale |

<details>
<summary>Full Image Node Listing (61 nodes)</summary>

| Node ID | Terraform Resource |
|---------|--------------------|
| `Flux2ProImageNode` | `comfyui_comfyui_flux2_pro_image_node` |
| `FluxKontextProImageNode` | `comfyui_comfyui_flux_kontext_pro_image_node` |
| `FluxProExpandNode` | `comfyui_comfyui_flux_pro_expand_node` |
| `FluxProFillNode` | `comfyui_comfyui_flux_pro_fill_node` |
| `FluxProUltraImageNode` | `comfyui_comfyui_flux_pro_ultra_image_node` |
| `BriaImageEditNode` | `comfyui_comfyui_bria_image_edit_node` |
| `BriaRemoveImageBackground` | `comfyui_comfyui_bria_remove_image_background` |
| `ByteDanceImageNode` | `comfyui_comfyui_byte_dance_image_node` |
| `ByteDanceSeedreamNode` | `comfyui_comfyui_byte_dance_seedream_node` |
| `GeminiImage2Node` | `comfyui_comfyui_gemini_image2_node` |
| `GeminiImageNode` | `comfyui_comfyui_gemini_image_node` |
| `GeminiNanoBanana2` | `comfyui_comfyui_gemini_nano_banana2` |
| `GrokImageEditNode` | `comfyui_comfyui_grok_image_edit_node` |
| `GrokImageNode` | `comfyui_comfyui_grok_image_node` |
| `HitPawGeneralImageEnhance` | `comfyui_comfyui_hit_paw_general_image_enhance` |
| `IdeogramV1` | `comfyui_comfyui_ideogram_v1` |
| `IdeogramV2` | `comfyui_comfyui_ideogram_v2` |
| `IdeogramV3` | `comfyui_comfyui_ideogram_v3` |
| `KlingImageGenerationNode` | `comfyui_comfyui_kling_image_generation_node` |
| `KlingOmniProImageNode` | `comfyui_comfyui_kling_omni_pro_image_node` |
| `KlingVirtualTryOnNode` | `comfyui_comfyui_kling_virtual_try_on_node` |
| `LumaImageModifyNode` | `comfyui_comfyui_luma_image_modify_node` |
| `LumaImageNode` | `comfyui_comfyui_luma_image_node` |
| `LumaReferenceNode` | `comfyui_comfyui_luma_reference_node` |
| `MagnificImageRelightNode` | `comfyui_comfyui_magnific_image_relight_node` |
| `MagnificImageSkinEnhancerNode` | `comfyui_comfyui_magnific_image_skin_enhancer_node` |
| `MagnificImageStyleTransferNode` | `comfyui_comfyui_magnific_image_style_transfer_node` |
| `MagnificImageUpscalerCreativeNode` | `comfyui_comfyui_magnific_image_upscaler_creative_node` |
| `MagnificImageUpscalerPreciseV2Node` | `comfyui_comfyui_magnific_image_upscaler_precise_v2_node` |
| `OpenAIDalle2` | `comfyui_comfyui_open_ai_dalle2` |
| `OpenAIDalle3` | `comfyui_comfyui_open_ai_dalle3` |
| `OpenAIGPTImage1` | `comfyui_comfyui_open_aigpt_image1` |
| `QuiverImageToSVGNode` | `comfyui_comfyui_quiver_image_to_svg_node` |
| `QuiverTextToSVGNode` | `comfyui_comfyui_quiver_text_to_svg_node` |
| `RecraftColorRGB` | `comfyui_comfyui_recraft_color_rgb` |
| `RecraftControls` | `comfyui_comfyui_recraft_controls` |
| `RecraftCreateStyleNode` | `comfyui_comfyui_recraft_create_style_node` |
| `RecraftCrispUpscaleNode` | `comfyui_comfyui_recraft_crisp_upscale_node` |
| `RecraftImageInpaintingNode` | `comfyui_comfyui_recraft_image_inpainting_node` |
| `RecraftImageToImageNode` | `comfyui_comfyui_recraft_image_to_image_node` |
| `RecraftRemoveBackgroundNode` | `comfyui_comfyui_recraft_remove_background_node` |
| `RecraftReplaceBackgroundNode` | `comfyui_comfyui_recraft_replace_background_node` |
| `RecraftStyleV3InfiniteStyleLibrary` | `comfyui_comfyui_recraft_style_v3_infinite_style_library` |
| `RecraftStyleV3RealisticImage` | `comfyui_comfyui_recraft_style_v3_realistic_image` |
| `RecraftTextToImageNode` | `comfyui_comfyui_recraft_text_to_image_node` |
| `RecraftTextToVectorNode` | `comfyui_comfyui_recraft_text_to_vector_node` |
| `RecraftV4TextToImageNode` | `comfyui_comfyui_recraft_v4_text_to_image_node` |
| `RecraftV4TextToVectorNode` | `comfyui_comfyui_recraft_v4_text_to_vector_node` |
| `RecraftVectorizeImageNode` | `comfyui_comfyui_recraft_vectorize_image_node` |
| `ReveImageCreateNode` | `comfyui_comfyui_reve_image_create_node` |
| `ReveImageEditNode` | `comfyui_comfyui_reve_image_edit_node` |
| `ReveImageRemixNode` | `comfyui_comfyui_reve_image_remix_node` |
| `RunwayTextToImageNode` | `comfyui_comfyui_runway_text_to_image_node` |
| `StabilityStableImageSD_3_5Node` | `comfyui_comfyui_stability_stable_image_sd_3_5_node` |
| `StabilityStableImageUltraNode` | `comfyui_comfyui_stability_stable_image_ultra_node` |
| `StabilityUpscaleConservativeNode` | `comfyui_comfyui_stability_upscale_conservative_node` |
| `StabilityUpscaleCreativeNode` | `comfyui_comfyui_stability_upscale_creative_node` |
| `StabilityUpscaleFastNode` | `comfyui_comfyui_stability_upscale_fast_node` |
| `TopazImageEnhance` | `comfyui_comfyui_topaz_image_enhance` |
| `WanImageToImageApi` | `comfyui_comfyui_wan_image_to_image_api` |
| `WanTextToImageApi` | `comfyui_comfyui_wan_text_to_image_api` |
| `WavespeedImageUpscaleNode` | `comfyui_comfyui_wavespeed_image_upscale_node` |

</details>

### Video (77 nodes, 17 providers)

Video is the largest partner node category — covering text-to-video, image-to-video,
video editing, lip sync, camera control, and video enhancement.

| Provider | Count | Capabilities |
|----------|-------|-------------|
| **Kling** | 22 | Text/image-to-video, camera control, lip sync, video effects, avatars, OmniPro series, video extend |
| **Vidu** | 13 | Text/image-to-video (V2 & V3), start-end frames, reference video, extend, multi-frame |
| **Wan** | 8 | Text/image-to-video (V1 & V2), reference video, video continuation, video edit |
| **ByteDance** | 4 | Text/image-to-video, first-last frame, image reference |
| **Grok** | 4 | Video generation, editing, extend, reference |
| **MiniMax** | 4 | Hailuo video, text/image/subject-to-video |
| **PixVerse** | 4 | Text/image-to-video, templates, transitions |
| **Luma** | 3 | Video generation, image-to-video, concepts |
| **Moonvalley Marey** | 3 | Text/image/video-to-video |
| **Runway** | 3 | Image-to-video Gen3a & Gen4, first-last frame |
| **LTXV** | 2 | Text/image-to-video |
| **Veo** | 2 | Video generation, Veo3 first-last frame |
| **Bria** | 1 | Video background removal |
| **HitPaw** | 1 | Video enhancement |
| **Sora** | 1 | OpenAI Sora 2 video generation |
| **Topaz** | 1 | Video enhancement |
| **WaveSpeed** | 1 | Flash VSR (video super-resolution) |

<details>
<summary>Full Video Node Listing (77 nodes)</summary>

| Node ID | Terraform Resource |
|---------|--------------------|
| `BriaRemoveVideoBackground` | `comfyui_comfyui_bria_remove_video_background` |
| `ByteDanceFirstLastFrameNode` | `comfyui_comfyui_byte_dance_first_last_frame_node` |
| `ByteDanceImageReferenceNode` | `comfyui_comfyui_byte_dance_image_reference_node` |
| `ByteDanceImageToVideoNode` | `comfyui_comfyui_byte_dance_image_to_video_node` |
| `ByteDanceTextToVideoNode` | `comfyui_comfyui_byte_dance_text_to_video_node` |
| `GrokVideoEditNode` | `comfyui_comfyui_grok_video_edit_node` |
| `GrokVideoExtendNode` | `comfyui_comfyui_grok_video_extend_node` |
| `GrokVideoNode` | `comfyui_comfyui_grok_video_node` |
| `GrokVideoReferenceNode` | `comfyui_comfyui_grok_video_reference_node` |
| `HitPawVideoEnhance` | `comfyui_comfyui_hit_paw_video_enhance` |
| `KlingAvatarNode` | `comfyui_comfyui_kling_avatar_node` |
| `KlingCameraControlI2VNode` | `comfyui_comfyui_kling_camera_control_i2_v_node` |
| `KlingCameraControlT2VNode` | `comfyui_comfyui_kling_camera_control_t2_v_node` |
| `KlingCameraControls` | `comfyui_comfyui_kling_camera_controls` |
| `KlingDualCharacterVideoEffectNode` | `comfyui_comfyui_kling_dual_character_video_effect_node` |
| `KlingFirstLastFrameNode` | `comfyui_comfyui_kling_first_last_frame_node` |
| `KlingImage2VideoNode` | `comfyui_comfyui_kling_image2_video_node` |
| `KlingImageToVideoWithAudio` | `comfyui_comfyui_kling_image_to_video_with_audio` |
| `KlingLipSyncAudioToVideoNode` | `comfyui_comfyui_kling_lip_sync_audio_to_video_node` |
| `KlingLipSyncTextToVideoNode` | `comfyui_comfyui_kling_lip_sync_text_to_video_node` |
| `KlingMotionControl` | `comfyui_comfyui_kling_motion_control` |
| `KlingOmniProEditVideoNode` | `comfyui_comfyui_kling_omni_pro_edit_video_node` |
| `KlingOmniProFirstLastFrameNode` | `comfyui_comfyui_kling_omni_pro_first_last_frame_node` |
| `KlingOmniProImageToVideoNode` | `comfyui_comfyui_kling_omni_pro_image_to_video_node` |
| `KlingOmniProTextToVideoNode` | `comfyui_comfyui_kling_omni_pro_text_to_video_node` |
| `KlingOmniProVideoToVideoNode` | `comfyui_comfyui_kling_omni_pro_video_to_video_node` |
| `KlingSingleImageVideoEffectNode` | `comfyui_comfyui_kling_single_image_video_effect_node` |
| `KlingStartEndFrameNode` | `comfyui_comfyui_kling_start_end_frame_node` |
| `KlingTextToVideoNode` | `comfyui_comfyui_kling_text_to_video_node` |
| `KlingTextToVideoWithAudio` | `comfyui_comfyui_kling_text_to_video_with_audio` |
| `KlingVideoExtendNode` | `comfyui_comfyui_kling_video_extend_node` |
| `KlingVideoNode` | `comfyui_comfyui_kling_video_node` |
| `LtxvApiImageToVideo` | `comfyui_comfyui_ltxv_api_image_to_video` |
| `LtxvApiTextToVideo` | `comfyui_comfyui_ltxv_api_text_to_video` |
| `LumaConceptsNode` | `comfyui_comfyui_luma_concepts_node` |
| `LumaImageToVideoNode` | `comfyui_comfyui_luma_image_to_video_node` |
| `LumaVideoNode` | `comfyui_comfyui_luma_video_node` |
| `MinimaxHailuoVideoNode` | `comfyui_comfyui_minimax_hailuo_video_node` |
| `MinimaxImageToVideoNode` | `comfyui_comfyui_minimax_image_to_video_node` |
| `MinimaxSubjectToVideoNode` | `comfyui_comfyui_minimax_subject_to_video_node` |
| `MinimaxTextToVideoNode` | `comfyui_comfyui_minimax_text_to_video_node` |
| `MoonvalleyImg2VideoNode` | `comfyui_comfyui_moonvalley_img2_video_node` |
| `MoonvalleyTxt2VideoNode` | `comfyui_comfyui_moonvalley_txt2_video_node` |
| `MoonvalleyVideo2VideoNode` | `comfyui_comfyui_moonvalley_video2_video_node` |
| `OpenAIVideoSora2` | `comfyui_comfyui_open_ai_video_sora2` |
| `PixverseImageToVideoNode` | `comfyui_comfyui_pixverse_image_to_video_node` |
| `PixverseTemplateNode` | `comfyui_comfyui_pixverse_template_node` |
| `PixverseTextToVideoNode` | `comfyui_comfyui_pixverse_text_to_video_node` |
| `PixverseTransitionVideoNode` | `comfyui_comfyui_pixverse_transition_video_node` |
| `RunwayFirstLastFrameNode` | `comfyui_comfyui_runway_first_last_frame_node` |
| `RunwayImageToVideoNodeGen3a` | `comfyui_comfyui_runway_image_to_video_node_gen3a` |
| `RunwayImageToVideoNodeGen4` | `comfyui_comfyui_runway_image_to_video_node_gen4` |
| `TopazVideoEnhance` | `comfyui_comfyui_topaz_video_enhance` |
| `Veo3FirstLastFrameNode` | `comfyui_comfyui_veo3_first_last_frame_node` |
| `VeoVideoGenerationNode` | `comfyui_comfyui_veo_video_generation_node` |
| `Vidu2ImageToVideoNode` | `comfyui_comfyui_vidu2_image_to_video_node` |
| `Vidu2ReferenceVideoNode` | `comfyui_comfyui_vidu2_reference_video_node` |
| `Vidu2StartEndToVideoNode` | `comfyui_comfyui_vidu2_start_end_to_video_node` |
| `Vidu2TextToVideoNode` | `comfyui_comfyui_vidu2_text_to_video_node` |
| `Vidu3ImageToVideoNode` | `comfyui_comfyui_vidu3_image_to_video_node` |
| `Vidu3StartEndToVideoNode` | `comfyui_comfyui_vidu3_start_end_to_video_node` |
| `Vidu3TextToVideoNode` | `comfyui_comfyui_vidu3_text_to_video_node` |
| `ViduExtendVideoNode` | `comfyui_comfyui_vidu_extend_video_node` |
| `ViduImageToVideoNode` | `comfyui_comfyui_vidu_image_to_video_node` |
| `ViduMultiFrameVideoNode` | `comfyui_comfyui_vidu_multi_frame_video_node` |
| `ViduReferenceVideoNode` | `comfyui_comfyui_vidu_reference_video_node` |
| `ViduStartEndToVideoNode` | `comfyui_comfyui_vidu_start_end_to_video_node` |
| `ViduTextToVideoNode` | `comfyui_comfyui_vidu_text_to_video_node` |
| `Wan2ImageToVideoApi` | `comfyui_comfyui_wan2_image_to_video_api` |
| `Wan2ReferenceVideoApi` | `comfyui_comfyui_wan2_reference_video_api` |
| `Wan2TextToVideoApi` | `comfyui_comfyui_wan2_text_to_video_api` |
| `Wan2VideoContinuationApi` | `comfyui_comfyui_wan2_video_continuation_api` |
| `Wan2VideoEditApi` | `comfyui_comfyui_wan2_video_edit_api` |
| `WanImageToVideoApi` | `comfyui_comfyui_wan_image_to_video_api` |
| `WanReferenceVideoApi` | `comfyui_comfyui_wan_reference_video_api` |
| `WanTextToVideoApi` | `comfyui_comfyui_wan_text_to_video_api` |
| `WavespeedFlashVSRNode` | `comfyui_comfyui_wavespeed_flash_vsr_node` |

</details>

### Audio (11 nodes, 2 providers)

Audio partner nodes cover text-to-speech, speech-to-text, voice cloning, audio
isolation, sound effects, and audio-to-audio transformation.

| Provider | Count | Capabilities |
|----------|-------|-------------|
| **ElevenLabs** | 8 | Text-to-speech, speech-to-speech, speech-to-text, text-to-dialogue, text-to-sound-effects, audio isolation, instant voice clone, voice selector |
| **Stability AI** | 3 | Text-to-audio, audio-to-audio, audio inpaint |

<details>
<summary>Full Audio Node Listing (11 nodes)</summary>

| Node ID | Terraform Resource |
|---------|--------------------|
| `ElevenLabsAudioIsolation` | `comfyui_comfyui_eleven_labs_audio_isolation` |
| `ElevenLabsInstantVoiceClone` | `comfyui_comfyui_eleven_labs_instant_voice_clone` |
| `ElevenLabsSpeechToSpeech` | `comfyui_comfyui_eleven_labs_speech_to_speech` |
| `ElevenLabsSpeechToText` | `comfyui_comfyui_eleven_labs_speech_to_text` |
| `ElevenLabsTextToDialogue` | `comfyui_comfyui_eleven_labs_text_to_dialogue` |
| `ElevenLabsTextToSoundEffects` | `comfyui_comfyui_eleven_labs_text_to_sound_effects` |
| `ElevenLabsTextToSpeech` | `comfyui_comfyui_eleven_labs_text_to_speech` |
| `ElevenLabsVoiceSelector` | `comfyui_comfyui_eleven_labs_voice_selector` |
| `StabilityAudioInpaint` | `comfyui_comfyui_stability_audio_inpaint` |
| `StabilityAudioToAudio` | `comfyui_comfyui_stability_audio_to_audio` |
| `StabilityTextToAudio` | `comfyui_comfyui_stability_text_to_audio` |

</details>

### Text (5 nodes, 2 providers)

Text partner nodes provide LLM chat and multi-modal input capabilities.

| Provider | Count | Capabilities |
|----------|-------|-------------|
| **OpenAI** | 3 | Chat completion, chat configuration, file inputs |
| **Gemini** | 2 | Chat completion, file inputs |

<details>
<summary>Full Text Node Listing (5 nodes)</summary>

| Node ID | Terraform Resource |
|---------|--------------------|
| `GeminiInputFiles` | `comfyui_comfyui_gemini_input_files` |
| `GeminiNode` | `comfyui_comfyui_gemini_node` |
| `OpenAIChatConfig` | `comfyui_comfyui_open_ai_chat_config` |
| `OpenAIChatNode` | `comfyui_comfyui_open_ai_chat_node` |
| `OpenAIInputFiles` | `comfyui_comfyui_open_ai_input_files` |

</details>

### 3D (26 nodes, 4 providers)

3D partner nodes generate 3D models from text or images, with rigging, texturing,
retopology, and format conversion capabilities.

| Provider | Count | Capabilities |
|----------|-------|-------------|
| **Tripo** | 8 | Text/image/multiview-to-model, refine, rig, retarget, texture, format conversion |
| **Meshy** | 7 | Text/image/multi-image-to-model, refine, rig, animate, texture |
| **Tencent (Hunyuan3D)** | 6 | Text/image-to-model, 3D parts, texture editing, smart topology, UV mapping |
| **Rodin** | 5 | Regular/detail/smooth/sketch generation, Gen2 |

<details>
<summary>Full 3D Node Listing (26 nodes)</summary>

| Node ID | Terraform Resource |
|---------|--------------------|
| `MeshyAnimateModelNode` | `comfyui_comfyui_meshy_animate_model_node` |
| `MeshyImageToModelNode` | `comfyui_comfyui_meshy_image_to_model_node` |
| `MeshyMultiImageToModelNode` | `comfyui_comfyui_meshy_multi_image_to_model_node` |
| `MeshyRefineNode` | `comfyui_comfyui_meshy_refine_node` |
| `MeshyRigModelNode` | `comfyui_comfyui_meshy_rig_model_node` |
| `MeshyTextToModelNode` | `comfyui_comfyui_meshy_text_to_model_node` |
| `MeshyTextureNode` | `comfyui_comfyui_meshy_texture_node` |
| `Rodin3D_Detail` | `comfyui_comfyui_rodin3_d_detail` |
| `Rodin3D_Gen2` | `comfyui_comfyui_rodin3_d_gen2` |
| `Rodin3D_Regular` | `comfyui_comfyui_rodin3_d_regular` |
| `Rodin3D_Sketch` | `comfyui_comfyui_rodin3_d_sketch` |
| `Rodin3D_Smooth` | `comfyui_comfyui_rodin3_d_smooth` |
| `Tencent3DPartNode` | `comfyui_comfyui_tencent3_d_part_node` |
| `Tencent3DTextureEditNode` | `comfyui_comfyui_tencent3_d_texture_edit_node` |
| `TencentImageToModelNode` | `comfyui_comfyui_tencent_image_to_model_node` |
| `TencentModelTo3DUVNode` | `comfyui_comfyui_tencent_model_to3_duv_node` |
| `TencentSmartTopologyNode` | `comfyui_comfyui_tencent_smart_topology_node` |
| `TencentTextToModelNode` | `comfyui_comfyui_tencent_text_to_model_node` |
| `TripoConversionNode` | `comfyui_comfyui_tripo_conversion_node` |
| `TripoImageToModelNode` | `comfyui_comfyui_tripo_image_to_model_node` |
| `TripoMultiviewToModelNode` | `comfyui_comfyui_tripo_multiview_to_model_node` |
| `TripoRefineNode` | `comfyui_comfyui_tripo_refine_node` |
| `TripoRetargetNode` | `comfyui_comfyui_tripo_retarget_node` |
| `TripoRigNode` | `comfyui_comfyui_tripo_rig_node` |
| `TripoTextToModelNode` | `comfyui_comfyui_tripo_text_to_model_node` |
| `TripoTextureNode` | `comfyui_comfyui_tripo_texture_node` |

</details>

## Terraform Resource Naming

Partner nodes follow the same naming convention as all generated node resources:

```
ComfyUI Node ID  →  comfyui_comfyui_<snake_case_name>
```

The snake_case conversion:
- `KlingTextToVideoNode` → `comfyui_comfyui_kling_text_to_video_node`
- `ElevenLabsTextToSpeech` → `comfyui_comfyui_eleven_labs_text_to_speech`
- `Rodin3D_Detail` → `comfyui_comfyui_rodin3_d_detail`

The double `comfyui_` prefix (provider name + resource prefix) is standard for
Terraform provider naming conventions.

## API Key Requirements

Partner nodes require API keys from their respective providers. Unlike core nodes
that only need the ComfyUI server connection, each partner service requires separate
authentication.

### Provider API Key Summary

| Provider | API Key Source | Environment Variable (proposed) |
|----------|---------------|-------------------------------|
| Kling | [Kling AI Platform](https://klingai.com/) | `KLING_API_KEY` |
| OpenAI | [OpenAI Platform](https://platform.openai.com/) | `OPENAI_API_KEY` |
| Stability AI | [Stability AI](https://platform.stability.ai/) | `STABILITY_API_KEY` |
| Recraft | [Recraft API](https://www.recraft.ai/) | `RECRAFT_API_KEY` |
| ElevenLabs | [ElevenLabs](https://elevenlabs.io/) | `ELEVENLABS_API_KEY` |
| BFL (Flux) | [Black Forest Labs](https://blackforestlabs.ai/) | `BFL_API_KEY` |
| Luma | [Luma AI](https://lumalabs.ai/) | `LUMA_API_KEY` |
| Gemini/Veo | [Google AI Studio](https://aistudio.google.com/) | `GOOGLE_API_KEY` |
| Tripo | [Tripo AI](https://www.tripo3d.ai/) | `TRIPO_API_KEY` |
| Meshy | [Meshy](https://www.meshy.ai/) | `MESHY_API_KEY` |
| Ideogram | [Ideogram](https://ideogram.ai/) | `IDEOGRAM_API_KEY` |
| Runway | [Runway](https://runwayml.com/) | `RUNWAY_API_KEY` |
| MiniMax | [MiniMax](https://www.minimax.io/) | `MINIMAX_API_KEY` |
| Vidu | [Vidu](https://www.vidu.com/) | `VIDU_API_KEY` |
| Grok | [xAI](https://x.ai/) | `XAI_API_KEY` |
| ByteDance | [Volcengine](https://www.volcengine.com/) | `BYTEDANCE_API_KEY` |
| Magnific | [Magnific AI](https://magnific.ai/) | `MAGNIFIC_API_KEY` |
| Rodin | [Hyper AI](https://hyper.ai/) | `RODIN_API_KEY` |
| Tencent | [Tencent Cloud](https://cloud.tencent.com/) | `TENCENT_API_KEY` |
| Bria | [Bria AI](https://bria.ai/) | `BRIA_API_KEY` |
| PixVerse | [PixVerse](https://pixverse.ai/) | `PIXVERSE_API_KEY` |
| Moonvalley | [Moonvalley](https://moonvalley.com/) | `MOONVALLEY_API_KEY` |
| HitPaw | [HitPaw](https://www.hitpaw.com/) | `HITPAW_API_KEY` |
| Topaz | [Topaz Labs](https://www.topazlabs.com/) | `TOPAZ_API_KEY` |
| WaveSpeed | [WaveSpeed AI](https://wavespeed.ai/) | `WAVESPEED_API_KEY` |
| Quiver | [Quiver](https://quiver.art/) | `QUIVER_API_KEY` |
| Reve | [Reve AI](https://reve.art/) | `REVE_API_KEY` |
| LTXV | [Lightricks](https://www.lightricks.com/) | `LTXV_API_KEY` |
| Wan | [Alibaba/Wan](https://wan.video/) | `WAN_API_KEY` |

> **Note**: API keys are typically configured within ComfyUI's settings UI or
> environment variables. The Terraform provider proxies these through the ComfyUI
> server — the provider itself does not call third-party APIs directly.

## Example Usage

### Image Generation with Recraft

```hcl
# Generate an image using Recraft V4
resource "comfyui_comfyui_recraft_v4_text_to_image_node" "hero_image" {
  prompt = "A futuristic cityscape at sunset, photorealistic"
  # Additional attributes depend on the node's schema
}

# Use the generated image in a workflow
resource "comfyui_workflow" "recraft_pipeline" {
  workflow_json = jsonencode({
    "1" = {
      class_type = "RecraftV4TextToImageNode"
      inputs = {
        prompt = "A futuristic cityscape at sunset, photorealistic"
      }
    }
  })
}
```

### Video Generation with Kling

```hcl
# Generate a video from text using Kling
resource "comfyui_workflow" "kling_video" {
  workflow_json = jsonencode({
    "1" = {
      class_type = "KlingTextToVideoNode"
      inputs = {
        prompt    = "A cat walking through a garden, cinematic"
        duration  = "5"
        mode      = "std"
      }
    }
  })
}
```

### Text-to-Speech with ElevenLabs

```hcl
# Generate speech from text using ElevenLabs
resource "comfyui_workflow" "tts_pipeline" {
  workflow_json = jsonencode({
    "1" = {
      class_type = "ElevenLabsVoiceSelector"
      inputs = {
        voice_name = "Rachel"
      }
    }
    "2" = {
      class_type = "ElevenLabsTextToSpeech"
      inputs = {
        text  = "Hello, welcome to our application."
        voice = ["1", 0]
      }
    }
  })
}
```

### 3D Model Generation with Tripo

```hcl
# Generate a 3D model from text using Tripo
resource "comfyui_workflow" "model_gen" {
  workflow_json = jsonencode({
    "1" = {
      class_type = "TripoTextToModelNode"
      inputs = {
        prompt = "A medieval castle"
      }
    }
    "2" = {
      class_type = "TripoTextureNode"
      inputs = {
        model = ["1", 0]
      }
    }
  })
}
```

## Relationship to Core Provider

Partner node resources are part of the same code generation pipeline as core node
resources. They are extracted from the ComfyUI source tree by the Python extractors
(`extract_v1_nodes.py` and `extract_v3_nodes.py`), merged into `node_specs.json`,
and generated into Go resource files by `cmd/generate/`.

The only distinguishing factor is the `category` field in `node_specs.json`:
- Core nodes: categories like `sampling`, `conditioning`, `loaders`, etc.
- Partner nodes: categories prefixed with `api node/` (e.g., `api node/video/Kling`)

This means partner nodes:
- Follow the same virtual/plan-only resource pattern as core nodes
- Are registered in the same `AllResources()` function
- Are updated automatically when the ComfyUI submodule is bumped

## Cross-References

- [00 — Overview and Architecture](00-overview-and-architecture.md) — Plugin Framework fundamentals
- [04 — Resource Implementation](04-resource-implementation.md) — How resources are structured
- [14 — Naming Conventions](14-naming-conventions-and-style.md) — Resource naming patterns
- [24 — ComfyUI Provider Mapping](24-comfyui-provider-mapping.md) — API-to-Terraform mapping
