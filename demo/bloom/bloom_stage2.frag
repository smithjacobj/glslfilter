#version 330 core
#extension GL_ARB_separate_shader_objects : enable
#extension GL_ARB_explicit_uniform_location : enable
#extension GL_ARB_shading_language_420pack : enable

layout(location = 0) in vec2 fragTexCoord;
layout(binding = 0, location = 1) uniform sampler2D inputTexture;
layout(binding = 1, location = 2) uniform sampler2D previousResult;
layout(location = 2) uniform double threshold;

layout(location = 0) out vec4 fragColor;

void main() {
  // we need to calculate the fixed values of the gaussian function and supply them in a uniform, otherwise we calculate the same values for every pixel
}