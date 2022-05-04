#version 330 core
#extension GL_ARB_separate_shader_objects : enable
#extension GL_ARB_explicit_uniform_location : enable
#extension GL_ARB_shading_language_420pack : enable

layout(location = 0) in vec2 fragTexCoord;
layout(binding = 0, location = 1) uniform sampler2D inputTexture;
layout(location = 2) uniform double threshold;

layout(location = 0) out vec4 fragColor;

double luminance(vec4 rgba) {
    return rgba.a * (rgba.r * 0.2126 + rgba.g * 0.7152 + rgba.b * 0.0722);
}

void main() {
  if (luminance(fragColor) > threshold) {
      fragColor = texture2D(inputTexture, fragTexCoord);
  } else {
      fragColor.a = 0;
  }
}