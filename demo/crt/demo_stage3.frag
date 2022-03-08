#version 330 core
#extension GL_ARB_separate_shader_objects : enable
#extension GL_ARB_explicit_uniform_location : enable
#extension GL_ARB_shading_language_420pack : enable

layout(location = 0) uniform ivec2 _;
layout(location = 1) in vec2 fragTexCoord;
layout(binding = 0) uniform sampler2D previousResult;

layout(location = 0) out vec4 fragColor;

const int kSampleOffset = 12;

void main() {
    vec2 resultSize = textureSize(previousResult, 0);
    vec2 texelSize = vec2(1.0 / resultSize.x, 1.0 / resultSize.y);
    vec4 glowColor = texture(previousResult, fragTexCoord);
    vec2 sampleTexCoord;
    for(int i = -kSampleOffset; i < kSampleOffset; i++) {
        sampleTexCoord.y = fragTexCoord.y + (texelSize.y * i);
        for(int j = -kSampleOffset; j < kSampleOffset; j++) {
            if(i == 0 && j == 0) {
                continue;
            }
            sampleTexCoord.x = fragTexCoord.x + (texelSize.x * j);

            glowColor = (glowColor + texture(previousResult, sampleTexCoord));
        }
    }
    glowColor /= kSampleOffset * kSampleOffset;
    fragColor = texture(previousResult, fragTexCoord);
    fragColor += (vec4(vec3(1.0) / vec3(clamp(fragColor.r + fragColor.g + fragColor.b, 0.9, 1.0)), 1.0) * glowColor) * 0.5;
}
