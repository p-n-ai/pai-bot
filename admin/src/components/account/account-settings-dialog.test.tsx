import { fireEvent, render, screen } from "@testing-library/react";
import * as React from "react";
import { vi } from "vitest";
import { AccountSettingsDialog } from "./account-settings-dialog";
import type { AuthUser } from "@/lib/api";

const linkedIdentitiesCardMock = vi.fn();

const DialogContext = React.createContext<{ open: boolean; onOpenChange: (open: boolean) => void } | null>(null);

function MockDialog({
  open,
  onOpenChange,
  children,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: React.ReactNode;
}) {
  return <DialogContext.Provider value={{ open, onOpenChange }}>{children}</DialogContext.Provider>;
}

function MockDialogTrigger({
  render,
  children,
}: {
  render: React.ReactElement;
  children?: React.ReactNode;
}) {
  const context = React.useContext(DialogContext);
  if (!context) return null;
  return React.cloneElement(render as React.ReactElement<{ onClick?: () => void; children?: React.ReactNode }>, {
    onClick: () => context.onOpenChange(true),
    children,
  });
}

function MockDialogContent({ children }: { children: React.ReactNode }) {
  const context = React.useContext(DialogContext);
  if (!context?.open) return null;
  return <div>{children}</div>;
}

function MockDialogSection({ children }: { children: React.ReactNode }) {
  return <div>{children}</div>;
}

vi.mock("@/components/ui/dialog", () => ({
  Dialog: MockDialog,
  DialogTrigger: MockDialogTrigger,
  DialogContent: MockDialogContent,
  DialogHeader: MockDialogSection,
  DialogTitle: MockDialogSection,
  DialogDescription: MockDialogSection,
  DialogFooter: ({ showCloseButton }: { showCloseButton?: boolean }) => <div data-show-close={String(Boolean(showCloseButton))} />,
}));

vi.mock("@/components/account/linked-identities-card", () => ({
  LinkedIdentitiesCard: (props: unknown) => {
    linkedIdentitiesCardMock(props);
    return <div>Linked identities surface</div>;
  },
}));

const currentUser: AuthUser = {
  user_id: "user-1",
  tenant_id: "tenant-a",
  tenant_slug: "school-a",
  tenant_name: "School A",
  role: "teacher",
  name: "Teacher A",
  email: "teacher@example.com",
};

describe("AccountSettingsDialog", () => {
  beforeEach(() => {
    linkedIdentitiesCardMock.mockClear();
  });

  it("opens the dialog and passes the current auth context into linked identities", () => {
    render(<AccountSettingsDialog currentUser={currentUser} nextPath="/dashboard" />);

    fireEvent.click(screen.getByRole("button", { name: /settings/i }));

    expect(screen.getByText("Account settings")).toBeInTheDocument();
    expect(screen.getByText("Linked identities surface")).toBeInTheDocument();
    expect(linkedIdentitiesCardMock).toHaveBeenCalledWith(
      expect.objectContaining({
        enabled: true,
        nextPath: "/dashboard",
      }),
    );
  });

  it("disables linked identities when no signed-in user is present", () => {
    render(<AccountSettingsDialog currentUser={null} nextPath="/dashboard" />);

    fireEvent.click(screen.getByRole("button", { name: /settings/i }));

    expect(linkedIdentitiesCardMock).toHaveBeenCalledWith(
      expect.objectContaining({
        enabled: false,
      }),
    );
  });
});
