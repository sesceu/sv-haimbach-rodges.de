# Blog Automation Agent Walkthrough

The agent is a local Go web application that helps you create blog posts with Gemini-powered text refinement and image generation.

## Prerequisites

- **Go**: Installed (you have it).
- **GitHub CLI (`gh`)**: Installed (you have it).
- **Gemini API Key**: You need an API key from [Google AI Studio](https://aistudio.google.com/).

## How to Run

1.  **Navigate to the agent directory**:
    ```bash
    cd agent
    ```

2.  **Build the agent** (if you haven't already):
    ```bash
    go build -o agent-bin
    ```

3.  **Run the agent**:
    You can pass the API key via a flag or environment variable.

    **Option A: Environment Variable**
    ```bash
    export GEMINI_API_KEY="your_api_key_here"
    ./agent-bin
    ```

    **Option B: Flag**
    ```bash
    ./agent-bin -key "your_api_key_here"
    ```

4.  **Open in Browser**:
    Go to [http://localhost:8080](http://localhost:8080).

## Workflow

1.  **Content**: Paste your rough text. Click "Refine with Gemini" to get a polished blog post.
2.  **Image**:
    *   **Upload**: Select an image from your computer.
    *   **Generate**: Enter a prompt and click "Generate" to use Gemini/Imagen.
3.  **Publish**:
    *   Enter a **Title**.
    *   Select a **Date**.
    *   Click **Create Post & PR**.

## What Happens Behind the Scenes

*   **Files**: A new Markdown file is created in `content/blog/` and the image is moved to `static/img/blog/`.
*   **Git**: A new branch `post/your-title-timestamp` is created.
*   **GitHub**: The branch is pushed, and a Pull Request is automatically created using `gh`.

## Troubleshooting

*   **Gemini Error**: If image generation fails, ensure your API key has access to the Imagen model.
*   **Git Error**: Ensure you have write access to the repository and `gh` is authenticated (`gh auth login`).
