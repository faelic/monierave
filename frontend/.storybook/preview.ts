import type { Preview } from "@storybook/nextjs-vite";
import "@fontsource-variable/fraunces";
import "@fontsource-variable/source-sans-3";

import "../src/app/globals.css";

const preview: Preview = {
  parameters: {
    a11y: {
      test: "error",
    },
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    layout: "centered",
  },
};

export default preview;
