#version 330 core
#extension GL_ARB_separate_shader_objects : enable
#extension GL_ARB_explicit_uniform_location : enable
#extension GL_ARB_shading_language_420pack : enable

layout(location = 0) uniform ivec2 _;
layout(location = 1) in vec2 fragTexCoord;
layout(binding = 0) uniform sampler2D previousResult;

layout(location = 0) out vec4 fragColor;

float M_PI = radians(180.0);

void main() {
    float scanlineValue = sin(gl_FragCoord.y * M_PI / 12.0) + 0.75;
    vec4 multColor = vec4(vec3(clamp(scanlineValue, 0, 1.0)), 1.0);
    fragColor = texture(previousResult, fragTexCoord) * multColor;
}
