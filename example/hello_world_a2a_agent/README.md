# Hello World A2A Agent

A simple hello world agent built with Google ADK that is provided by [a2a_root](https://github.com/google/adk-python/tree/main/contributing/samples/a2a_root).

## What it does

This agent can:

- Roll dice with any number of sides
- Check if numbers are prime
- Track roll history in state

## Prerequisites

Set the `GOOGLE_API_KEY` environment variable:

```bash
export GOOGLE_API_KEY=your-api-key
```

## Running the agent

Install dependencies:

```bash
pip install -r requirements.txt
```

Run the agent:

```bash
uvicorn main:a2a_app --host localhost --port 8001
```

## Running with Docker

Build the image:

```bash
docker build -t hello-world-a2a-agent .
```

Run the container:

```bash
docker run -e GOOGLE_API_KEY=your-api-key -p 8001:8001 hello-world-a2a-agent
```

## Available tools

- `roll_die(sides: int)` - Rolls a die with specified sides and returns the result
- `check_prime(nums: list[int])` - Checks if given numbers are prime

## Example queries

- "Roll an 8-sided die"
- "Roll a 20-sided die and check if the result is prime"
- "Check if 17, 25, and 31 are prime numbers"
