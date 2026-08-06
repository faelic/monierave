import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import { Input } from "@/components/ui/input";

const meta = {
  title: "Foundation/Input",
  component: Input,
  args: {
    "aria-label": "Account number",
    inputMode: "numeric",
    placeholder: "10-digit account number",
  },
  tags: ["autodocs"],
} satisfies Meta<typeof Input>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};

export const Invalid: Story = {
  args: {
    "aria-invalid": true,
    defaultValue: "123",
  },
};

export const Disabled: Story = {
  args: {
    disabled: true,
  },
};
