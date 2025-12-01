大模型网关
===

# 项目说明

通过标准的OpenAI、Gemini、Claude API 格式请求转换为中间格式，转发后端配置的大模型。

# 架构

客户端到服务端请求路径

```plaintext
clientMessage -> GeneralMessage -> serverMessage
```

```plaintext
openai -> openai
openai -> gemini
openai -> claude

gemini -> openai
gemini -> gemini
gemini -> claude

claude -> openai
claude -> gemini
claude -> claude
```