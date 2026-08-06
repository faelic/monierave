import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import { Button } from "@/components/ui/button";

const meta = {
  title: "Foundation/Button",
  component: Button,
  args: {
    children: "Continue to review",
  },
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
} satisfies Meta<typeof Button>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Primary: Story = {};

export const Secondary: Story = {
  args: {
    variant: "secondary",
  },
};

export const Danger: Story = {
  args: {
    children: "Close account",
    variant: "danger",
  },
};

export const Loading: Story = {
  args: {
    children: "Sending transfer",
    loading: true,
  },
};
