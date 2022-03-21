#version 330 core
#extension GL_ARB_separate_shader_objects : enable
#extension GL_ARB_explicit_uniform_location : enable
#extension GL_ARB_shading_language_420pack : enable

layout(location = 0) in vec2 fragTexCoord;
layout(location = 1) uniform ivec2 outputResolution;
layout(binding = 0, location = 2) uniform sampler2D inputTexture;
layout(binding = 1, location = 3) uniform sampler2D pixelTexture;

layout(location = 0) out vec4 fragColor;

const float M_PI = radians(180.0);

vec2 barrelDistort(vec2 v) {
  const float kStretchRatio = 1.1;

  vec2 uv = ((fragTexCoord.xy) - vec2(0.5)) / kStretchRatio;
  float uva = atan(uv.x, uv.y);
  float uvd = sqrt(dot(uv, uv));
  //k = negative for pincushion, positive for barrel
  float k = 0.1;
  uvd = uvd*(1.0 + k*uvd*uvd);
  return vec2(0.5) + vec2(sin(uva), cos(uva))*uvd;
}

vec4 pixelate(const vec4 inputColor, const vec2 texCoord, const vec2 inputTexelSize, const vec2 inputTextureSize) {
  const vec2 pixelTextureSize = textureSize(pixelTexture, 0);
  const vec2 pixelTexelSize = 1.0 / pixelTextureSize;

  const float columnNumber = texCoord.x / inputTexelSize.x;
  const bool isOddColumn = (floor(mod(columnNumber, 2.0)) > 0.0);

  vec2 modifiedTexCoord = texCoord * inputTextureSize;
  if (isOddColumn) {
    modifiedTexCoord.y += 0.5;
  }

  return inputColor * texture(pixelTexture, modifiedTexCoord);
}

float scanlineValue(const vec2 texCoord, const vec2 inputTexelSize) {
  return clamp(sin(texCoord.y / inputTexelSize.y * M_PI) * 0.25 + 1.0, 0, 1.0);
}

vec4 sample(const vec2 texCoord, const vec2 inputTexelSize, const vec2 inputTextureSize) {
  const vec4 initialColor = texture(inputTexture, texCoord);
  const vec4 pixelatedColor = pixelate(initialColor, texCoord, inputTexelSize, inputTextureSize);
  const vec4 scanlineColor = pixelatedColor * vec4(vec3(scanlineValue(texCoord, inputTexelSize)), 1.0);

  return scanlineColor;
}

void main() {
  const vec2 inputTextureSize = textureSize(inputTexture, 0);
  const vec2 inputTexelSize = 1.0 / inputTextureSize;
  const vec2 fragUVSize = 1.0 / outputResolution;
  const vec2 distortedFragTexCoord = barrelDistort(fragTexCoord);

  fragColor = vec4(0);
  // const vec2 supersampleOffsets[1] = {vec2(0)};
  const vec2 supersampleOffsets[16] = {
    vec2(0, 0), vec2(0, 0.25), vec2(0, 0.5), vec2(0, 0.75),
    vec2(0.25, 0), vec2(0.25, 0.25), vec2(0.25, 0.5), vec2(0.25, 0.75),
    vec2(0.5, 0), vec2(0.5, 0.25), vec2(0.5, 0.5), vec2(0.5, 0.75),
    vec2(0.75, 0), vec2(0.75, 0.25), vec2(0.75, 0.5), vec2(0.75, 0.75),
  };
  for (int i = 0; i < supersampleOffsets.length(); i++) {
    const vec2 supersampleTexCoord = distortedFragTexCoord + supersampleOffsets[i] * fragUVSize;
    fragColor += sample(supersampleTexCoord, inputTexelSize, inputTextureSize);
  }
  fragColor /= supersampleOffsets.length();
}
