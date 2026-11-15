# Welcome to Your New Wails3 Project!

Congratulations on generating your Wails3 application! This README will guide you through the next steps to get your project up and running.

## Getting Started

1. Navigate to your project directory in the terminal.

2. To run your application in development mode, use the following command:

   ```
   wails3 dev
   ```

   This will start your application and enable hot-reloading for both frontend and backend changes.

3. To build your application for production, use:

   ```
   wails3 build
   ```

   This will create a production-ready executable in the `build` directory.

## Releasing

Pushing a `v*` tag builds a universal macOS DMG and attaches it to a GitHub Release
([workflow](.github/workflows/release-macos.yml)):

```
git tag v1.2.0 && git push origin v1.2.0
```

- The tag sets the app version (`v1.2.0` → `1.2.0`). Tags with a hyphen, like `v1.2.0-beta.1`, become prereleases.
- Running the workflow manually from the Actions tab builds the DMG as a downloadable artifact without publishing a Release.
- The app is ad-hoc signed, not notarized. On first launch, right-click the app and choose **Open**, or run
  `xattr -dr com.apple.quarantine /Applications/dotmd.app`.

To build the DMG locally: `wails3 task darwin:package:universal && wails3 task darwin:create:dmg` (output in `bin/`).

## Exploring Wails3 Features

Now that you have your project set up, it's time to explore the features that Wails3 offers:

1. **Check out the examples**: The best way to learn is by example. Visit the `examples` directory in the `v3/examples` directory to see various sample applications.

2. **Run an example**: To run any of the examples, navigate to the example's directory and use:

   ```
   go run .
   ```

   Note: Some examples may be under development during the alpha phase.

3. **Explore the documentation**: Visit the [Wails3 documentation](https://v3.wails.io/) for in-depth guides and API references.

4. **Join the community**: Have questions or want to share your progress? Join the [Wails Discord](https://discord.gg/JDdSxwjhGf) or visit the [Wails discussions on GitHub](https://github.com/wailsapp/wails/discussions).

## Project Structure

Take a moment to familiarize yourself with your project structure:

- `frontend/`: Contains your frontend code (HTML, CSS, JavaScript/TypeScript)
- `main.go`: The entry point of your Go backend
- `app.go`: Define your application structure and methods here
- `wails.json`: Configuration file for your Wails project

## Next Steps

1. Modify the frontend in the `frontend/` directory to create your desired UI.
2. Add backend functionality in `main.go`.
3. Use `wails3 dev` to see your changes in real-time.
4. When ready, build your application with `wails3 build`.

Happy coding with Wails3! If you encounter any issues or have questions, don't hesitate to consult the documentation or reach out to the Wails community.
