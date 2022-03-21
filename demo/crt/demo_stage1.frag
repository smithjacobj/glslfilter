#version 330 core
#extension GL_ARB_separate_shader_objects : enable
#extension GL_ARB_explicit_uniform_location : enable
#extension GL_ARB_shading_language_420pack : enable

layout(location = 0) in vec2 fragTexCoord;
layout(location = 1) in vec2 viewportSize;
layout(binding = 1) uniform sampler2D baseTexture;
layout(binding = 2) uniform sampler2D tileTexture;

layout(location = 0) out vec4 fragColor;

void main() {
  vec2 baseTextureSize = textureSize(baseTexture, 0);
  vec2 tileTextureSize = textureSize(tileTexture, 0);
  float textureRatioX = tileTextureSize.x / (viewportSize.x / baseTextureSize.x);
  float tileTextureColumn = (gl_FragCoord.x / tileTextureSize.x * textureRatioX);
  float tileTextureColumnMod = floor(mod(tileTextureColumn, 2.0));
  float tileTexelSizeY = 1.0 / tileTextureSize.y;
  vec2 tileTextureFragCoord = fragTexCoord * baseTextureSize;
  if (tileTextureColumnMod == 1.0) {
    tileTextureFragCoord.y += tileTextureSize.y * tileTexelSizeY / 2.0;
  }
  fragColor = texture(baseTexture, fragTexCoord) * texture(tileTexture, tileTextureFragCoord);
}
