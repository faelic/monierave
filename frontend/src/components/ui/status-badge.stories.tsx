import type { Meta, StoryObj } from "@storybook/nextjs-vite";

import { StatusBadge } from "@/components/ui/status-badge";

const meta = {
  title: "Foundation/Status Badge",
  component: StatusBadge,
  args: {
    children: "Active",
  },
  tags: ["autodocs"],
} satisfies Meta<typeof StatusBadge>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Active: Story = {
  args: {
    tone: "positive",
  },
};

export const Pending: Story = {
  args: {
    children: "Pending",
    tone: "info",
  },
};

export const Frozen: Story = {
  args: {
    children: "Frozen",
    tone: "warning",
  },
};

export const Failed: Story = {
  args: {
    children: "Failed",
    tone: "danger",
  },
};
