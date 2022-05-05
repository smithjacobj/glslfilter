#version 330 core
#extension GL_ARB_separate_shader_objects : require
#extension GL_ARB_explicit_uniform_location : require
#extension GL_ARB_shading_language_420pack : require

#define MAX_COUNT 16

layout(location = 0) in vec2 fragTexCoord;
layout(location = 1) uniform ivec2 outputResolution;
layout(binding = 0, location = 2) uniform sampler2D inputTexture;
layout(location = 3) uniform int count;
layout(std140, binding = 1, location = 4) uniform Colors {
    vec3 filters[MAX_COUNT];
} colorFilters;
layout(std140, binding = 2, location = 5) uniform Offsets {
    ivec2 offsets[MAX_COUNT];
} colorDivergencePx;

layout(location = 0) out vec4 fragColor;

const vec2 pixelSize = vec2(1) / outputResolution;

vec4 getDivergentColor(vec3 color, ivec2 offset) {
    const vec2 uvOffset = -1 * offset * pixelSize;
    return vec4(color, 1) * texture(inputTexture, fragTexCoord - uvOffset);
}

void main() {
    if (colorFilters.filters.length() != colorDivergencePx.offsets.length()) {
        // TODO: indicate some sort of error
    } else {
        fragColor = vec4(0);
        for (int i = 0; i < count; i++) {
            fragColor += getDivergentColor(colorFilters.filters[i], colorDivergencePx.offsets[i]);
        }
    }
}