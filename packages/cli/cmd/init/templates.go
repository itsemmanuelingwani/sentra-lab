package init

import "fmt"

func generateLabYAML(name string) string {
	return fmt.Sprintf(`# Sentra Lab Configuration
name: %s
version: "1.0"

# Agent configuration
agent:
  runtime: python  # python, nodejs, go
  entry_point: agent.py
  timeout: 30s

# Mock services
mocks:
  openai:
    enabled: true
    port: 8080
    latency_ms: 1000
    rate_limit: 3500  # requests per minute
    error_rate: 0.01  # 1%% random errors
  
  stripe:
    enabled: true
    port: 8081
    latency_ms: 500
  
  coreledger:
    enabled: true
    port: 8082

# Simulation settings
simulation:
  record_full_trace: true
  enable_cost_tracking: true
  max_concurrent_scenarios: 10

# Storage
storage:
  recordings_dir: .sentra-lab/recordings
  database: .sentra-lab/sentra.db
`, name)
}

func generateMocksYAML() string {
	return `# Mock Service Configuration
mocks:
  openai:
    models:
      - gpt-4
      - gpt-4-turbo-preview
      - gpt-3.5-turbo
    
    behavior:
      latency:
        min_ms: 500
        max_ms: 3000
        distribution: normal
      
      rate_limiting:
        enabled: true
        rpm: 3500
        tpm: 90000
      
      errors:
        - type: rate_limit_exceeded
          probability: 0.01
          status_code: 429
        
        - type: service_unavailable
          probability: 0.005
          status_code: 503
    
    fixtures:
      responses: fixtures/openai-responses.yaml
      patterns: fixtures/openai-patterns.yaml

  stripe:
    behavior:
      latency:
        min_ms: 100
        max_ms: 1000
        distribution: normal
      
      webhooks:
        enabled: true
        delivery_delay_ms: 1000
    
    fixtures:
      cards: fixtures/stripe-cards.yaml
      errors: fixtures/stripe-errors.yaml

  coreledger:
    behavior:
      latency:
        min_ms: 200
        max_ms: 800
    
    fixtures:
      agents: fixtures/coreledger-agents.yaml
`
}

func generateBasicScenario(name string) string {
	return fmt.Sprintf(`# Basic Test Scenario
name: "Basic Agent Test"
description: "Verify agent responds to simple input"
version: "1.0"

variables:
  user_input: "Hello, can you help me?"

steps:
  - id: "agent-initialization"
    action: verify_agent_ready
    expect:
      - status: ready
      - timeout: 5s
  
  - id: "simple-request"
    action: agent_request
    input: "{{user_input}}"
    expect:
      - status: success
      - response_time: <10s
      - response_not_empty: true
  
  - id: "verify-no-errors"
    action: assert
    conditions:
      - no_errors: true
      - execution_time: <15s

# Project: %s
`, name)
}

func generateOpenAIScenario() string {
	return `# OpenAI API Test
name: "OpenAI Integration Test"
description: "Test agent's interaction with OpenAI mock"
version: "1.0"

steps:
  - id: "openai-request"
    action: agent_request
    input: "What is 2+2?"
    expect:
      - calls: ["openai.chat.completions"]
      - response_contains: "4"
  
  - id: "verify-cost"
    action: verify_cost
    expect:
      - total_cost: <$0.10
      - openai_tokens: <500
  
  - id: "rate-limit-handling"
    action: inject_error
    service: openai
    error: rate_limit_exceeded
    probability: 1.0
  
  - id: "agent-retry"
    action: agent_request
    input: "What is 3+3?"
    expect:
      - retry_count: >0
      - backoff_strategy: exponential
      - final_status: success
`
}

func generatePaymentScenario() string {
	return `# Payment Flow Test
name: "Payment Processing"
description: "Test complete payment flow with Stripe"
version: "1.0"

variables:
  amount: 99.99
  currency: "usd"
  customer_email: "test@example.com"

steps:
  - id: "create-payment-intent"
    action: agent_request
    input: "Process payment for ${{amount}}"
    expect:
      - calls: ["stripe.payment_intents.create"]
      - payment_status: "requires_payment_method"
  
  - id: "attach-payment-method"
    action: mock_response
    service: stripe
    endpoint: /v1/payment_methods
    response:
      id: "pm_test_visa"
      type: "card"
      card:
        brand: "visa"
        last4: "4242"
  
  - id: "confirm-payment"
    action: agent_request
    input: "Confirm payment"
    expect:
      - calls: ["stripe.payment_intents.confirm"]
      - payment_status: "succeeded"
  
  - id: "verify-webhook"
    action: verify_webhook
    service: stripe
    event_type: "payment_intent.succeeded"
    timeout: 5s
`
}

func generateGitignore() string {
	return `.sentra-lab/recordings/
.sentra-lab/sentra.db
.sentra-lab/cache/

__pycache__/
*.py[cod]
*$py.class
*.so
.Python
env/
venv/
ENV/

node_modules/
dist/
build/
*.log

.DS_Store
.idea/
.vscode/
*.swp
*.swo
`
}

func generateReadme(name string) string {
	return fmt.Sprintf(`# %s

Sentra Lab project for testing AI agents locally.

## Getting Started

1. Start mock services:
   \'\'\'bash
   sentra lab start
   \'\'\'

2. Run test scenarios:
   \'\'\'bash
   sentra lab test
   \'\'\'

3. Replay failed tests:
   \'\'\'bash
   sentra lab replay
   \'\'\'

## Project Structure

- \`scenarios/\` - Test scenarios (YAML)
- \`fixtures/\` - Mock response fixtures
- \`tests/\` - Additional test files
- \`.sentra-lab/\` - Recordings and database

## Documentation

- [Sentra Lab Docs](https://docs.sentra.dev)
- [Writing Scenarios](https://docs.sentra.dev/scenarios)
- [Mock Services](https://docs.sentra.dev/mocks)

## Support

- GitHub: https://github.com/sentra-lab/sentra-lab
- Discord: https://discord.gg/sentra-lab
`, name)
}

func generatePythonAgent() string {
	return `#!/usr/bin/env python3
"""
Example Python agent using OpenAI.
"""
import os
from openai import OpenAI

# Use Sentra Lab mock endpoint
client = OpenAI(
    api_key="mock_key_123",
    base_url=os.getenv("OPENAI_API_BASE", "http://localhost:8080/v1")
)

def handle_query(user_input: str) -> str:
    """Process user query using OpenAI."""
    response = client.chat.completions.create(
        model="gpt-4",
        messages=[
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": user_input}
        ],
        temperature=0.7,
        max_tokens=500
    )
    
    return response.choices[0].message.content

if __name__ == "__main__":
    print("Agent ready. Enter 'quit' to exit.")
    
    while True:
        user_input = input("\nYou: ").strip()
        
        if user_input.lower() == "quit":
            break
        
        if not user_input:
            continue
        
        try:
            response = handle_query(user_input)
            print(f"\nAgent: {response}")
        except Exception as e:
            print(f"\nError: {e}")
`
}

func generatePythonRequirements() string {
	return `openai>=1.0.0
requests>=2.31.0
pydantic>=2.0.0
pytest>=7.4.0
pytest-asyncio>=0.21.0
`
}

func generateNodeAgent() string {
	return `import OpenAI from 'openai';

// Use Sentra Lab mock endpoint
const client = new OpenAI({
  apiKey: 'mock_key_123',
  baseURL: process.env.OPENAI_API_BASE || 'http://localhost:8080/v1'
});

async function handleQuery(userInput: string): Promise<string> {
  const response = await client.chat.completions.create({
    model: 'gpt-4',
    messages: [
      { role: 'system', content: 'You are a helpful assistant.' },
      { role: 'user', content: userInput }
    ],
    temperature: 0.7,
    max_tokens: 500
  });

  return response.choices[0].message.content || '';
}

async function main() {
  console.log('Agent ready. Enter "quit" to exit.');
  
  const readline = require('readline');
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout
  });

  rl.on('line', async (input: string) => {
    if (input.toLowerCase() === 'quit') {
      rl.close();
      return;
    }

    if (!input.trim()) {
      return;
    }

    try {
      const response = await handleQuery(input);
      console.log(\`\nAgent: \${response}\`);
    } catch (error) {
      console.error(\`\nError: \${error}\`);
    }
  });
}

main();
`
}

func generatePackageJSON(name string) string {
	return fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "description": "Sentra Lab agent",
  "main": "agent.ts",
  "scripts": {
    "start": "ts-node agent.ts",
    "test": "jest"
  },
  "dependencies": {
    "openai": "^4.0.0"
  },
  "devDependencies": {
    "@types/node": "^20.0.0",
    "ts-node": "^10.9.0",
    "typescript": "^5.0.0"
  }
}
`, name)
}

func generateTSConfig() string {
	return `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "commonjs",
    "lib": ["ES2020"],
    "outDir": "./dist",
    "rootDir": "./",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true
  },
  "include": ["*.ts"],
  "exclude": ["node_modules", "dist"]
}
`
}

func generateGoAgent() string {
	return `package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sashabaranov/go-openai"
)

func main() {
	// Use Sentra Lab mock endpoint
	config := openai.DefaultConfig("mock_key_123")
	baseURL := os.Getenv("OPENAI_API_BASE")
	if baseURL == "" {
		baseURL = "http://localhost:8080/v1"
	}
	config.BaseURL = baseURL

	client := openai.NewClientWithConfig(config)

	fmt.Println("Agent ready. Enter 'quit' to exit.")

	for {
		fmt.Print("\nYou: ")
		var input string
		fmt.Scanln(&input)

		if input == "quit" {
			break
		}

		if input == "" {
			continue
		}

		response, err := handleQuery(client, input)
		if err != nil {
			fmt.Printf("\nError: %v\n", err)
			continue
		}

		fmt.Printf("\nAgent: %s\n", response)
	}
}

func handleQuery(client *openai.Client, userInput string) (string, error) {
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful assistant.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userInput,
				},
			},
			Temperature: 0.7,
			MaxTokens:   500,
		},
	)

	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}
`
}

func generateGoMod(name string) string {
	return fmt.Sprintf(`module %s

go 1.21

require github.com/sashabaranov/go-openai v1.17.0
`, name)
}
