package init

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Scaffolder struct {
	projectDir string
	template   string
}

func NewScaffolder(projectDir, template string) *Scaffolder {
	return &Scaffolder{
		projectDir: projectDir,
		template:   template,
	}
}

func (s *Scaffolder) InitGit() error {
	cmd := exec.Command("git", "init")
	cmd.Dir = s.projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git init failed: %w", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = s.projectDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit from Sentra Lab")
	cmd.Dir = s.projectDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	return nil
}

func (s *Scaffolder) CreateFixtures() error {
	fixturesDir := filepath.Join(s.projectDir, "fixtures")

	fixtures := map[string]string{
		"openai-responses.yaml": generateOpenAIFixtures(),
		"openai-patterns.yaml":  generateOpenAIPatterns(),
		"stripe-cards.yaml":     generateStripeCards(),
		"stripe-errors.yaml":    generateStripeErrors(),
	}

	if s.template == "fullstack" {
		fixtures["coreledger-agents.yaml"] = generateCoreLedgerAgents()
	}

	for filename, content := range fixtures {
		path := filepath.Join(fixturesDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write fixture %s: %w", filename, err)
		}
	}

	return nil
}

func generateOpenAIFixtures() string {
	return `# OpenAI Mock Response Fixtures
responses:
  - pattern: "what is.*"
    model: gpt-4
    response: |
      Let me help you with that question. Based on my knowledge, I can provide you with a detailed answer.
    tokens:
      prompt: 20
      completion: 25
  
  - pattern: "hello|hi|hey"
    model: gpt-4
    response: |
      Hello! How can I assist you today?
    tokens:
      prompt: 10
      completion: 12
  
  - pattern: ".*math.*|.*calculate.*"
    model: gpt-4
    response: |
      I'll help you with that calculation. Let me work through this step by step.
    tokens:
      prompt: 15
      completion: 30

  - pattern: ".*code.*|.*program.*"
    model: gpt-4
    response: |
      Here's a code example that addresses your question:
      
      ` + "```python" + `
      def example_function():
          return "Hello, World!"
      ` + "```" + `
    tokens:
      prompt: 25
      completion: 45

default:
  model: gpt-4
  response: |
    I understand your question. Let me provide a comprehensive response based on the information available.
  tokens:
    prompt: 20
    completion: 30
`
}

func generateOpenAIPatterns() string {
	return `# OpenAI Pattern Matching Rules
patterns:
  greetings:
    - hello
    - hi
    - hey
    - good morning
    - good afternoon
    - good evening
  
  questions:
    - what is
    - what are
    - how do
    - how can
    - why is
    - why are
    - where is
    - when is
  
  math:
    - calculate
    - compute
    - solve
    - "\\d+\\s*[+\\-*/]\\s*\\d+"
  
  coding:
    - write code
    - write a program
    - write a function
    - create a script
    - implement

response_strategies:
  greetings:
    tone: friendly
    length: short
  
  questions:
    tone: informative
    length: medium
    include_examples: true
  
  math:
    tone: precise
    include_calculation_steps: true
  
  coding:
    tone: technical
    include_code_blocks: true
    language: auto-detect
`
}

func generateStripeCards() string {
	return `# Stripe Test Card Numbers
cards:
  valid:
    - number: "4242424242424242"
      brand: visa
      exp_month: 12
      exp_year: 2025
      cvc: "123"
      description: "Standard Visa - Always succeeds"
    
    - number: "5555555555554444"
      brand: mastercard
      exp_month: 12
      exp_year: 2025
      cvc: "123"
      description: "Standard Mastercard - Always succeeds"
    
    - number: "378282246310005"
      brand: amex
      exp_month: 12
      exp_year: 2025
      cvc: "1234"
      description: "American Express - Always succeeds"
  
  declined:
    - number: "4000000000000002"
      brand: visa
      error: card_declined
      decline_code: generic_decline
      description: "Generic decline"
    
    - number: "4000000000009995"
      brand: visa
      error: card_declined
      decline_code: insufficient_funds
      description: "Insufficient funds"
    
    - number: "4000000000009987"
      brand: visa
      error: card_declined
      decline_code: lost_card
      description: "Lost card"
    
    - number: "4000000000009979"
      brand: visa
      error: card_declined
      decline_code: stolen_card
      description: "Stolen card"
  
  special:
    - number: "4000002500003155"
      brand: visa
      behavior: require_3ds
      description: "Requires 3D Secure authentication"
    
    - number: "4000000000000341"
      brand: visa
      behavior: processing_error
      description: "Charge succeeds but fails to capture"
`
}

func generateStripeErrors() string {
	return `# Stripe Error Scenarios
errors:
  rate_limit:
    status_code: 429
    type: rate_limit_error
    message: "Too many requests hit the API too quickly."
    retry_after: 5
  
  card_errors:
    - code: card_declined
      decline_code: generic_decline
      message: "Your card was declined."
    
    - code: card_declined
      decline_code: insufficient_funds
      message: "Your card has insufficient funds."
    
    - code: expired_card
      message: "Your card has expired."
    
    - code: incorrect_cvc
      message: "Your card's security code is incorrect."
    
    - code: processing_error
      message: "An error occurred while processing your card."
  
  api_errors:
    - type: invalid_request_error
      message: "Invalid request parameters."
      param: amount
    
    - type: authentication_error
      message: "Invalid API key provided."
    
    - type: api_error
      message: "An error occurred on Stripe's end."
`
}

func generateCoreLedgerAgents() string {
	return `# CoreLedger Agent Fixtures
agents:
  - id: agent_001
    name: "Payment Processor"
    type: payment
    balance_usd: 1000.00
    transaction_limit: 500.00
    status: active
  
  - id: agent_002
    name: "Data Analyst"
    type: analytics
    balance_usd: 500.00
    transaction_limit: 100.00
    status: active
  
  - id: agent_003
    name: "Customer Support"
    type: support
    balance_usd: 250.00
    transaction_limit: 50.00
    status: active

policies:
  payment:
    max_transaction: 500.00
    daily_limit: 5000.00
    require_approval: false
  
  analytics:
    max_transaction: 100.00
    daily_limit: 1000.00
    require_approval: false
  
  support:
    max_transaction: 50.00
    daily_limit: 500.00
    require_approval: true
`
}
