# FunCallArchitect

FunCallArchitect is an LLM-powered function calling framework designed to interpret user queries and execute appropriate function calls to retrieve information. It provides a structured approach to handling complex requests by breaking them down into a series of nested function calls.

## Core Functionality

- **Query Interpretation**: Analyzes user requests and determines the necessary function calls to fulfill them.
- **Function Call Planning**: Generates a structure of nested function calls to address user queries.
- **Execution Orchestration**: Manages the execution of planned function calls, including handling dependencies between functions.
- **Tool Integration**: Allows integration of custom tools and functions to extend the system's capabilities.
- **Progress Tracking**: Provides real-time updates on the execution process.

## Key Components

1. **Agent**: The high-level abstraction for processing user requests.
2. **RequestHandler**: Manages the process of interpreting user messages and executing appropriate actions.
3. **Orchestrator**: Handles the execution context for function calls, including memoization for efficiency.
4. **LLM Integration**: Utilizes language models for query interpretation and function call planning.
5. **Tools**: A flexible system for defining and managing available functions and their specifications.

## Features

- JSON Schema generation for constrained output
- Server implementation for handling user requests via REST API and Server-Sent Events (SSE)
- Support for concurrent function execution
- Memoization of function results for improved performance
- Customizable progress tracking and logging

## Use Cases

FunCallArchitect can be applied to various scenarios where user queries need to be broken down into a series of function calls, such as:

- Information retrieval systems
- Task automation
- Query processing for databases or APIs
- Building conversational AI systems

This project aims to provide a playground for developers to build LLM-powered applications that can understand and act on user requests through a structured function calling approach.
