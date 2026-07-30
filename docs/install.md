# Building from Source

## Prerequisites

Before building ytmusic-tui from source, ensure you have the following prerequisites installed:

- [Go](https://go.dev/dl/) version 1.25 or higher
- Git

## Installation Steps

1. Clone the repository:

   ```bash
   git clone https://github.com/kumneger0/ytmusic-tui.git
   cd ytmusic-tui
   ```

2. Create a Spotify Application:

   1. Go to the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard/applications) and log in.
   2. Click on **"Create an app"**.
   3. Give your application a name and description.
   4. Find the **"Redirect URIs"** section and add the following URI:
      ```
      http://127.0.0.1:9292
      ```
   5. Save your changes.
   6. Now, view your application's settings and find your **Client ID** and **Client Secret**.
   7. Set them as environment variables. For example, you can add the following lines to your shell's configuration file (e.g., `.bashrc`, `.zshrc`):
      ```bash
      export SPOTIFY_CLIENT_ID="<your-client-id>"
      export SPOTIFY_CLIENT_SECRET="<your-client-secret>"
      ```
   8. Remember to replace `<your-client-id>` and `<your-client-secret>` with the actual credentials from your Spotify application.

3. Build the project with your Spotify credentials:

   ```bash
   make build
   ```

   > **Note for Linux Users:**
   > You must have a C compiler installed (e.g., `gcc`) and enable CGO to build the project successfully on Linux, as the audio library depends on C libraries. This is not required for Windows or macOS.
   > ```bash
   > CGO_ENABLED=1 make build
   > ```

If you encounter any issues during installation:  
For additional help, please [open an issue](https://github.com/kumneger0/ytmusic-tui/issues/new) on GitHub.
