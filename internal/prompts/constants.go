// Package prompts contains all prompt strings and descriptions used by the tools.
package prompts

// Task tool prompts
const (
	// TaskToolDescription is the description for the Task tool
	TaskToolDescription = `Launch a new agent that has access to the following tools: Bash, Glob, Grep, LS, exit_plan_mode, Read, Edit, MultiEdit, Write, NotebookRead, NotebookEdit, WebFetch, TodoRead, TodoWrite, WebSearch. When you are searching for a keyword or file and are not confident that you will find the right match in the first few tries, use the Agent tool to perform the search for you.

When to use the Agent tool:

- If you are searching for a keyword like "config" or "logger", or for questions like "which file does X?", the Agent tool is strongly recommended

When NOT to use the Agent tool:

- If you want to read a specific file path, use the Read or Glob tool instead of the Agent tool, to find the match more quickly
- If you are searching for a specific class definition like "class Foo", use the Glob tool instead, to find the match more quickly
- If you are searching for code within a specific file or set of 2-3 files, use the Read tool instead of the Agent tool, to find the match more quickly
- Writing code and running bash commands (use other tools for that)

Usage notes:

1. Launch multiple agents concurrently whenever possible, to maximize performance; to do that, use a single message with multiple tool uses
2. When the agent is done, it will return a single message back to you. The result returned by the agent is not visible to the user. To show the user the result, you should send a text message back to the user with a concise summary of the result.
3. Each agent invocation is stateless. You will not be able to send additional messages to the agent, nor will the agent be able to communicate with you outside of its final report. Therefore, your prompt should contain a highly detailed task description for the agent to perform autonomously and you should specify exactly what information the agent should return back to you in its final and only message to you.
4. The agent's outputs should generally be trusted
5. Clearly tell the agent whether you expect it to write code or just to do research (search, file reads, web fetches, etc.), since it is not aware of the user's intent`

	// TaskSearchResponseTemplate is the template for search task responses
	TaskSearchResponseTemplate = `Agent Report: Search Task Completed

Task: %s

Search Results:
- Performed comprehensive search using Grep and Glob tools
- Examined file structure using LS tool
- Read relevant files to understand context
- Identified key locations and patterns

Key Findings:
- Located relevant files and code sections
- Analyzed patterns and implementations
- Documented search results with file paths and line numbers

Recommendations:
- Review the identified files for detailed implementation
- Consider the context and relationships between components
- Use the specific file paths provided for further investigation

Note: This is a simulated agent response. In a real implementation, this would contain actual search results and file analysis.`

	// TaskAnalysisResponseTemplate is the template for analysis task responses
	TaskAnalysisResponseTemplate = `Agent Report: Analysis Task Completed

Task: %s

Analysis Summary:
- Examined codebase structure and organization
- Analyzed implementation patterns and conventions
- Reviewed code quality and architectural decisions
- Identified key components and their relationships

Technical Findings:
- Code follows established patterns and best practices
- Architecture is well-structured with clear separation of concerns
- Dependencies are properly managed and organized
- Security considerations are appropriately addressed

Recommendations:
- Current implementation is solid and follows good practices
- Consider the insights provided for future development
- Review specific code sections for detailed understanding

Note: This is a simulated agent response. In a real implementation, this would contain actual code analysis and specific findings.`

	// TaskExplorationResponseTemplate is the template for exploration task responses
	TaskExplorationResponseTemplate = `Agent Report: Exploration Task Completed

Task: %s

Exploration Results:
- Mapped directory structure and file organization
- Identified key modules and their purposes
- Analyzed relationships between components
- Documented important files and their roles

Structure Overview:
- Project follows standard Go conventions
- Clear separation between internal and public APIs
- Well-organized package structure
- Comprehensive documentation and examples

Key Discoveries:
- Located main entry points and configuration
- Identified core functionality and extensions
- Found relevant tests and documentation
- Mapped dependencies and interfaces

Note: This is a simulated agent response. In a real implementation, this would contain actual exploration results and specific findings.`

	// TaskGenericResponseTemplate is the template for generic task responses
	TaskGenericResponseTemplate = `Agent Report: Task Completed

Task: %s

Execution Summary:
- Analyzed the requested task requirements
- Used appropriate tools to gather information
- Processed and analyzed the collected data
- Compiled findings into a comprehensive report

Results:
- Task completed successfully using available tools
- Information gathered and processed as requested
- Analysis performed according to specifications
- Report generated with relevant findings

Conclusion:
- The requested task has been completed
- All available information has been processed
- Results are ready for review and further action

Note: This is a simulated agent response. In a real implementation, this would contain actual task execution results and specific outputs.`
)

// Web tool prompts
const (
	// WebFetchToolDescription is the description for the WebFetch tool
	WebFetchToolDescription = `- Fetches content from a specified URL and processes it using an AI model
- Takes a URL and a prompt as input
- Fetches the URL content, converts HTML to markdown
- Processes the content with the prompt using a small, fast model
- Returns the model's response about the content
- Use this tool when you need to retrieve and analyze web content

Usage notes:
  - IMPORTANT: If an MCP-provided web fetch tool is available, prefer using that tool instead of this one, as it may have fewer restrictions. All MCP-provided tools start with "mcp__".
  - The URL must be a fully-formed valid URL
  - HTTP URLs will be automatically upgraded to HTTPS
  - The prompt should describe what information you want to extract from the page
  - This tool is read-only and does not modify any files
  - Results may be summarized if the content is very large
  - Includes a self-cleaning 15-minute cache for faster responses when repeatedly accessing the same URL`

	// WebSearchToolDescription is the description for the WebSearch tool
	WebSearchToolDescription = `- Allows Claude to search the web and use the results to inform responses
- Provides up-to-date information for current events and recent data
- Returns search result information formatted as search result blocks
- Use this tool for accessing information beyond Claude's knowledge cutoff
- Searches are performed automatically within a single API call

Usage notes:
  - Domain filtering is supported to include or block specific websites
  - Web search is only available in the US`

	// WebFetchContentProcessingTemplate is the template for processing web content with a prompt
	WebFetchContentProcessingTemplate = `**Processing request:** %s

**Content:**

%s

**Analysis:** Based on the prompt '%s', the content above has been fetched and converted from HTML to markdown format for easier reading.`
)

// Workflow tool prompts
const (
	// ExitPlanModeToolDescription is the description for the exit_plan_mode tool
	ExitPlanModeToolDescription = `Use this tool when you are in plan mode and have finished presenting your plan and are ready to code. This will prompt the user to exit plan mode. 
IMPORTANT: Only use this tool when the task requires planning the implementation steps of a task that requires writing code. For research tasks where you're gathering information, searching files, reading files or in general trying to understand the codebase - do NOT use this tool.

Eg. 
1. Initial task: "Search for and understand the implementation of vim mode in the codebase" - Do not use the exit plan mode tool because you are not planning the implementation steps of a task.
2. Initial task: "Help me implement yank mode for vim" - Use the exit plan mode tool after you have finished planning the implementation steps of the task.`

	// ExitPlanModeOutputTemplate is the template for exit plan mode output
	ExitPlanModeOutputTemplate = `## Plan Ready for Implementation

%s

---

**Ready to proceed with implementation?** This plan outlines the steps needed to complete your request. Please review and let me know if you'd like me to begin implementing these changes or if you'd like to modify the plan first.`
)

// Todo tool prompts
const (
	// TodoReadToolDescription is the description for the TodoRead tool
	TodoReadToolDescription = `Use this tool to read the current to-do list for the session. This tool should be used proactively and frequently to ensure that you are aware of
the status of the current task list. You should make use of this tool as often as possible, especially in the following situations:
- At the beginning of conversations to see what's pending
- Before starting new tasks to prioritize work
- When the user asks about previous tasks or plans
- Whenever you're uncertain about what to do next
- After completing tasks to update your understanding of remaining work
- After every few messages to ensure you're on track

Usage:
- This tool takes in no parameters. So leave the input blank or empty. DO NOT include a dummy object, placeholder string or a key like "input" or "empty". LEAVE IT BLANK.
- Returns a list of todo items with their status, priority, and content
- Use this information to track progress and plan next steps
- If no todos exist yet, an empty list will be returned`

	// TodoWriteToolDescription is the description for the TodoWrite tool
	TodoWriteToolDescription = `Use this tool to create and manage a structured task list for your current coding session. This helps you track progress, organize complex tasks, and demonstrate thoroughness to the user.
It also helps the user understand the progress of the task and overall progress of their requests.

## When to Use This Tool
Use this tool proactively in these scenarios:

1. Complex multi-step tasks - When a task requires 3 or more distinct steps or actions
2. Non-trivial and complex tasks - Tasks that require careful planning or multiple operations
3. User explicitly requests todo list - When the user directly asks you to use the todo list
4. User provides multiple tasks - When users provide a list of things to be done (numbered or comma-separated)
5. After receiving new instructions - Immediately capture user requirements as todos
6. When you start working on a task - Mark it as in_progress BEFORE beginning work. Ideally you should only have one todo as in_progress at a time
7. After completing a task - Mark it as completed and add any new follow-up tasks discovered during implementation

## When NOT to Use This Tool

Skip using this tool when:
1. There is only a single, straightforward task
2. The task is trivial and tracking it provides no organizational benefit
3. The task can be completed in less than 3 trivial steps
4. The task is purely conversational or informational

NOTE that you should not use this tool if there is only one trivial task to do. In this case you are better off just doing the task directly.

## Task States and Management

1. **Task States**: Use these states to track progress:
   - pending: Task not yet started
   - in_progress: Currently working on (limit to ONE task at a time)
   - completed: Task finished successfully

2. **Task Management**:
   - Update task status in real-time as you work
   - Mark tasks complete IMMEDIATELY after finishing (don't batch completions)
   - Only have ONE task in_progress at any time
   - Complete current tasks before starting new ones
   - Remove tasks that are no longer relevant from the list entirely

3. **Task Completion Requirements**:
   - ONLY mark a task as completed when you have FULLY accomplished it
   - If you encounter errors, blockers, or cannot finish, keep the task as in_progress
   - When blocked, create a new task describing what needs to be resolved
   - Never mark a task as completed if:
     - Tests are failing
     - Implementation is partial
     - You encountered unresolved errors
     - You couldn't find necessary files or dependencies

4. **Task Breakdown**:
   - Create specific, actionable items
   - Break complex tasks into smaller, manageable steps
   - Use clear, descriptive task names

When in doubt, use this tool. Being proactive with task management demonstrates attentiveness and ensures you complete all requirements successfully.`
)
