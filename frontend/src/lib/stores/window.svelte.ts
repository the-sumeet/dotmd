import { Window } from "@wailsio/runtime";

// In native macOS fullscreen the traffic lights are hidden, so nothing needs
// to be reserved for them.
class WindowStore {
  fullscreen = $state(false);

  async refresh() {
    try {
      this.fullscreen = await Window.IsFullscreen();
    } catch (err) {
      console.error("Failed to read fullscreen state", err);
    }
  }
}

export const windowState = new WindowStore();
