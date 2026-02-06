---
name: new-openai-sdk-app
description: Create and setup a new OpenAI Agents SDK application with interactive guidance for language choice, agent type selection (Basic, Voice, Realtime), project setup, and automatic verification.
---

You are tasked with helping the user create a new OpenAI Agents SDK application. Follow these steps carefully:

## Reference Documentation

Before starting, review the official documentation to ensure you provide accurate and up-to-date guidance. Use WebFetch to read these pages:

1. **Start with the overview**:
   - Python: https://openai.github.io/openai-agents-python/
   - TypeScript: https://openai.github.io/openai-agents-js/

2. **Based on the user's language and agent type choice, read the appropriate SDK reference**:
   - Python Quickstart: https://openai.github.io/openai-agents-python/quickstart/
   - TypeScript Quickstart: https://openai.github.io/openai-agents-js/guides/quickstart
   - Voice Agents (Python): https://openai.github.io/openai-agents-python/voice/quickstart/
   - Voice Agents (TypeScript): https://openai.github.io/openai-agents-js/guides/voice-agents/quickstart
   - Realtime Agents (Python): https://openai.github.io/openai-agents-python/realtime/quickstart/

3. **Read relevant guides mentioned in the overview** such as:
   - Agents and Running Agents
   - Tools and Function Tools
   - Handoffs for multi-agent orchestration
   - Guardrails for input/output validation
   - Sessions for conversation history
   - Tracing for debugging and monitoring
   - MCP (Model Context Protocol) integration

**IMPORTANT**: Always check for and use the latest versions of packages. Use WebSearch or WebFetch to verify current versions before installation.

## Gather Requirements

IMPORTANT: Ask these questions one at a time. Wait for the user's response before asking the next question. This makes it easier for the user to respond.

Ask the questions in this order (skip any that the user has already provided via arguments):

1. **Language** (ask first): "Would you like to use TypeScript or Python?"

   - Wait for response before continuing

2. **Project name** (ask second): "What would you like to name your project?"

   - If $ARGUMENTS is provided, use that as the project name and skip this question
   - Wait for response before continuing

3. **Agent type** (ask third): "What type of agent would you like to create?

   - **Basic Agent**: Standard text-based agent for chat, coding assistance, or automation tasks
   - **Voice Agent**: Speech-to-text and text-to-speech pipeline for voice interactions
   - **Realtime Agent**: Low-latency voice conversations using OpenAI's Realtime API (Beta)"

   - Wait for response before continuing

4. **Agent purpose** (ask fourth, but skip if #3 was sufficiently detailed): "What kind of agent are you building? Some examples:

   - Coding agent (SRE, security review, code review)
   - Business agent (customer support, content creation)
   - Custom agent (describe your use case)"
   - Wait for response before continuing

5. **Starting point** (ask fifth): "Would you like:

   - A minimal 'Hello World' example to start
   - A basic agent with common features (tools, handoffs)
   - A specific example based on your use case"
   - Wait for response before continuing

6. **Tooling choice** (ask sixth): Let the user know what tools you'll use, and confirm with them that these are the tools they want to use (for example, they may prefer pnpm or bun over npm for TypeScript, or poetry over pip for Python). Respect the user's preferences when executing on the requirements.

After all questions are answered, proceed to create the setup plan.

## Setup Plan

Based on the user's answers, create a plan that includes:

1. **Project initialization**:

   - Create project directory (if it doesn't exist)
   - Initialize package manager:
     - TypeScript: `npm init -y` and setup `package.json` with type: "module" and scripts (include a "typecheck" script)
     - Python: Create `requirements.txt` or use `poetry init`
   - Add necessary configuration files:
     - TypeScript: Create `tsconfig.json` with proper settings for the SDK
     - Python: Optionally create config files if needed

2. **Check for Latest Versions**:

   - BEFORE installing, use WebSearch or check npm/PyPI to find the latest version
   - For TypeScript: Check https://www.npmjs.com/package/@openai/agents
   - For Python: Check https://pypi.org/project/openai-agents/
   - Inform the user which version you're installing

3. **SDK Installation**:

   - TypeScript:
     - Basic: `npm install @openai/agents@latest`
     - Voice: `npm install @openai/agents@latest` (voice is included in the main package)
     - Realtime: `npm install @openai/agents@latest` (realtime is included in the main package)
   - Python:
     - Basic: `pip install openai-agents`
     - Voice: `pip install 'openai-agents[voice]'` (includes sounddevice, numpy dependencies)
     - Realtime: `pip install openai-agents` (realtime is included in the main package)
   - After installation, verify the installed version:
     - TypeScript: Check package.json or run `npm list @openai/agents`
     - Python: Run `pip show openai-agents`

4. **Create starter files**:

   Based on the agent type, create appropriate starter code:

   ### Basic Agent (TypeScript)
   ```typescript
   import { Agent, run } from '@openai/agents';

   const agent = new Agent({
     name: 'Assistant',
     instructions: 'You are a helpful assistant.',
   });

   async function main() {
     const result = await run(agent, 'Hello! How can you help me?');
     console.log(result.finalOutput);
   }

   main().catch(console.error);
   ```

   ### Basic Agent (Python)
   ```python
   from agents import Agent, Runner

   agent = Agent(
       name="Assistant",
       instructions="You are a helpful assistant."
   )

   result = Runner.run_sync(agent, "Hello! How can you help me?")
   print(result.final_output)
   ```

   ### Voice Agent (Python)
   ```python
   import asyncio
   import numpy as np
   import sounddevice as sd

   from agents import Agent, function_tool
   from agents.voice import AudioInput, SingleAgentVoiceWorkflow, VoicePipeline

   @function_tool
   def get_weather(city: str) -> str:
       """Get the weather for a given city."""
       return f"The weather in {city} is sunny."

   agent = Agent(
       name="VoiceAssistant",
       instructions="You are a helpful voice assistant. Be concise.",
       tools=[get_weather],
   )

   async def main():
       pipeline = VoicePipeline(workflow=SingleAgentVoiceWorkflow(agent))

       # For demo: 3 seconds of silence (replace with actual microphone input)
       buffer = np.zeros(24000 * 3, dtype=np.int16)
       audio_input = AudioInput(buffer=buffer)

       result = await pipeline.run(audio_input)

       player = sd.OutputStream(samplerate=24000, channels=1, dtype=np.int16)
       player.start()

       async for event in result.stream():
           if event.type == "voice_stream_event_audio":
               player.write(event.data)

   if __name__ == "__main__":
       asyncio.run(main())
   ```

   ### Voice Agent (TypeScript)
   ```typescript
   import { Agent, tool } from '@openai/agents';
   import { RealtimeAgent, RealtimeSession, OpenAIRealtimeWebSocket } from '@openai/agents/realtime';
   import { z } from 'zod';

   const getWeather = tool({
     name: 'get_weather',
     description: 'Get the weather for a given city',
     parameters: z.object({
       city: z.string().describe('The city to get weather for'),
     }),
     execute: async ({ city }) => {
       return `The weather in ${city} is sunny.`;
     },
   });

   const agent = new RealtimeAgent({
     name: 'VoiceAssistant',
     instructions: 'You are a helpful voice assistant. Be concise.',
     tools: [getWeather],
   });

   async function main() {
     const transport = new OpenAIRealtimeWebSocket();
     const session = new RealtimeSession(agent, { transport });

     await session.connect();
     console.log('Voice session connected! Ready for audio input.');

     // Handle session events
     session.on('audio', (event) => {
       // Process audio output
       console.log('Received audio chunk');
     });

     session.on('error', (event) => {
       console.error('Error:', event.error);
     });
   }

   main().catch(console.error);
   ```

   ### Realtime Agent (Python)
   ```python
   import asyncio
   from agents.realtime import RealtimeAgent, RealtimeRunner

   agent = RealtimeAgent(
       name="RealtimeAssistant",
       instructions="You are a helpful voice assistant. Keep responses brief and conversational.",
   )

   async def main():
       runner = RealtimeRunner(
           starting_agent=agent,
           config={
               "model_settings": {
                   "model_name": "gpt-realtime",
                   "voice": "ash",
                   "modalities": ["audio"],
                   "input_audio_format": "pcm16",
                   "output_audio_format": "pcm16",
                   "input_audio_transcription": {"model": "gpt-4o-mini-transcribe"},
                   "turn_detection": {"type": "semantic_vad", "interrupt_response": True},
               }
           },
       )

       session = await runner.run()

       async with session:
           print("Realtime session started! Streaming audio responses...")
           async for event in session:
               if event.type == "agent_start":
                   print(f"Agent started: {event.agent.name}")
               elif event.type == "audio":
                   # Handle audio output
                   pass
               elif event.type == "error":
                   print(f"Error: {event.error}")

   if __name__ == "__main__":
       asyncio.run(main())
   ```

   ### Realtime Agent (TypeScript)
   ```typescript
   import { RealtimeAgent, RealtimeSession, OpenAIRealtimeWebSocket } from '@openai/agents/realtime';

   const agent = new RealtimeAgent({
     name: 'RealtimeAssistant',
     instructions: 'You are a helpful voice assistant. Keep responses brief and conversational.',
   });

   async function main() {
     const transport = new OpenAIRealtimeWebSocket({
       model: 'gpt-4o-realtime-preview',
     });

     const session = new RealtimeSession(agent, {
       transport,
       config: {
         voice: 'ash',
         modalities: ['audio'],
         inputAudioFormat: 'pcm16',
         outputAudioFormat: 'pcm16',
         turnDetection: { type: 'semantic_vad', interruptResponse: true },
       },
     });

     await session.connect();
     console.log('Realtime session connected!');

     session.on('agent_start', (event) => {
       console.log(`Agent started: ${event.agent.name}`);
     });

     session.on('audio', (event) => {
       // Handle audio output chunks
       console.log('Received audio chunk');
     });

     session.on('error', (event) => {
       console.error('Error:', event.error);
     });
   }

   main().catch(console.error);
   ```

5. **Environment setup**:

   - Create a `.env.example` file with `OPENAI_API_KEY=your_api_key_here`
   - Add `.env` to `.gitignore`
   - Explain how to get an API key from https://platform.openai.com/api-keys

6. **Optional: Create .sgai directory structure**:
   - Offer to create `.sgai/` directory for agents, commands, and settings
   - Ask if they want any example multi-agent setups or custom tools

## Implementation

After gathering requirements and getting user confirmation on the plan:

1. Check for latest package versions using WebSearch or WebFetch
2. Execute the setup steps
3. Create all necessary files
4. Install dependencies (always use latest stable versions)
5. Verify installed versions and inform the user
6. Create a working example based on their agent type
7. Add helpful comments in the code explaining what each part does
8. **VERIFY THE CODE WORKS BEFORE FINISHING**:
   - For TypeScript:
     - Run `npx tsc --noEmit` to check for type errors
     - Fix ALL type errors until types pass completely
     - Ensure imports and types are correct
     - Only proceed when type checking passes with no errors
   - For Python:
     - Verify imports are correct
     - Check for basic syntax errors with `python -m py_compile <file>`
   - **DO NOT consider the setup complete until the code verifies successfully**

## Verification

After all files are created and dependencies are installed, use the appropriate verifier agent to validate that the Agent SDK application is properly configured and ready for use:

1. **For TypeScript projects**: Launch the **openai-sdk-verifier-ts** agent to validate the setup
2. **For Python projects**: Launch the **openai-sdk-verifier-py** agent to validate the setup
3. The agent will check SDK usage, configuration, functionality, and adherence to official documentation
4. Review the verification report and address any issues

## Getting Started Guide

Once setup is complete and verified, provide the user with:

1. **Next steps**:

   - How to set their API key: `export OPENAI_API_KEY=your_api_key_here`
   - How to run their agent:
     - TypeScript: `npx ts-node index.ts` or `npm start`
     - Python: `python main.py`

2. **Useful resources**:

   - Link to Python SDK docs: https://openai.github.io/openai-agents-python/
   - Link to TypeScript SDK docs: https://openai.github.io/openai-agents-js/
   - Explain key concepts: Agents, Handoffs, Guardrails, Sessions, Tracing

3. **Common next steps**:
   - How to add custom tools using `@function_tool` (Python) or `tool()` (TypeScript)
   - How to implement handoffs for multi-agent workflows
   - How to add guardrails for input/output validation
   - How to enable tracing for debugging
   - How to use MCP (Model Context Protocol) servers

## Important Notes

- **ALWAYS USE LATEST VERSIONS**: Before installing any packages, check for the latest versions using WebSearch or by checking npm/PyPI directly
- **VERIFY CODE RUNS CORRECTLY**:
  - For TypeScript: Run `npx tsc --noEmit` and fix ALL type errors before finishing
  - For Python: Verify syntax and imports are correct with `python -m py_compile`
  - Do NOT consider the task complete until the code passes verification
- Verify the installed version after installation and inform the user
- Check the official documentation for any version-specific requirements:
  - Python: 3.9 or higher
  - Node.js: 18 or higher (for TypeScript)
- Always check if directories/files already exist before creating them
- Use the user's preferred package manager (npm, yarn, pnpm, bun for TypeScript; pip, poetry for Python)
- Ensure all code examples are functional and include proper error handling
- Use modern syntax and patterns that are compatible with the latest SDK version
- Make the experience interactive and educational
- **ASK QUESTIONS ONE AT A TIME** - Do not ask multiple questions in a single response

## Key SDK Concepts to Explain

1. **Agents**: LLMs equipped with instructions and tools
2. **Handoffs**: Allow agents to delegate to other agents for specific tasks
3. **Guardrails**: Enable validation of agent inputs and outputs
4. **Sessions**: Automatically maintain conversation history across agent runs
5. **Tracing**: Built-in tracing for visualization, debugging, and monitoring
6. **Tools**: Turn any function into a tool with automatic schema generation

Begin by asking the FIRST requirement question only. Wait for the user's answer before proceeding to the next question.
