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
    vec2 uv = ((fragTexCoord.xy) - vec2(0.5)) / 1.1;
	float uva = atan(uv.x, uv.y);
    float uvd = sqrt(dot(uv, uv));
    //k = negative for pincushion, positive for barrel
    float k = 0.1;
    uvd = uvd*(1.0 + k*uvd*uvd);
    fragColor = texture(previousResult, vec2(0.5) + vec2(sin(uva), cos(uva))*uvd);
}
