import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { UserManagementPanel } from "@/components/user-management-panel";
import type { UserManagementView } from "@/lib/api";

const { issueInvite, reissueInvite } = vi.hoisted(() => ({
  issueInvite: vi.fn(),
  reissueInvite: vi.fn(),
}));

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>();
  return {
    ...actual,
    issueInvite,
    reissueInvite,
  };
});

const data: UserManagementView = {
  summary: {
    teachers: 1,
    parents: 1,
    pending_invites: 1,
    total_users: 3,
  },
  active_users: [
    {
      id: "admin-1",
      name: "Admin User",
      email: "admin@example.com",
      role: "admin",
      status: "active",
      created_at: "2026-04-06T10:00:00Z",
    },
    {
      id: "teacher-1",
      name: "Teacher One",
      email: "teacher@example.com",
      role: "teacher",
      status: "active",
      created_at: "2026-04-05T10:00:00Z",
    },
    {
      id: "parent-1",
      name: "Parent One",
      email: "parent@example.com",
      role: "parent",
      status: "active",
      created_at: "2026-04-04T10:00:00Z",
    },
  ],
  pending_invites: [
    {
      id: "invite-1",
      email: "newteacher@example.com",
      role: "teacher",
      status: "pending",
      delivery_status: "sent",
      expires_at: "2026-04-13T10:00:00Z",
      created_at: "2026-04-06T10:00:00Z",
      invited_by: "Admin User",
    },
  ],
};

describe("UserManagementPanel", () => {
  const writeText = vi.fn();

  beforeEach(() => {
    writeText.mockReset();
    Object.assign(navigator, {
      clipboard: {
        writeText,
      },
    });
  });

  it("renders summary counts and active users", () => {
    render(<UserManagementPanel data={data} />);

    expect(screen.getByText("Teachers")).toBeInTheDocument();
    expect(screen.getByText("Teacher One")).toBeInTheDocument();
    expect(screen.getByText("Parent One")).toBeInTheDocument();
  });

  it("filters active users by search text", () => {
    render(<UserManagementPanel data={data} />);

    fireEvent.change(screen.getByLabelText("Search users"), { target: { value: "teacher" } });

    expect(screen.getByText("Teacher One")).toBeInTheDocument();
    expect(screen.queryByText("Parent One")).not.toBeInTheDocument();
  });

  it("issues an invite and shows the activation link", async () => {
    issueInvite.mockResolvedValue({
      email: "newteacher@example.com",
      invite_token: "invite-token",
      activation_url: "http://localhost:3000/activate?token=invite-token",
      delivery_status: "sent",
    });

    render(<UserManagementPanel data={data} />);

    fireEvent.click(screen.getByRole("button", { name: "Invite user" }));
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "newteacher@example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Send invite" }));

    await waitFor(() =>
      expect(issueInvite).toHaveBeenCalledWith({
        email: "newteacher@example.com",
        role: "teacher",
      }),
    );
    expect(screen.getByLabelText("Latest activation link")).toHaveValue("http://localhost:3000/activate?token=invite-token");
    expect(screen.getByText("Invite email sent to newteacher@example.com.")).toBeInTheDocument();
  });

  it("reissues a pending invite and shows the fresh activation link", async () => {
    reissueInvite.mockResolvedValue({
      email: "newteacher@example.com",
      invite_token: "invite-token-reissued",
      activation_url: "http://localhost:3000/activate?token=invite-token-reissued",
      delivery_status: "sent",
    });

    render(<UserManagementPanel data={data} />);

    fireEvent.click(screen.getByRole("tab", { name: "Pending invites" }));
    fireEvent.click(screen.getByRole("button", { name: "Resend email" }));

    await waitFor(() => expect(reissueInvite).toHaveBeenCalledWith("invite-1"));
    expect(screen.getByLabelText("Latest activation link")).toHaveValue("http://localhost:3000/activate?token=invite-token-reissued");
  });

  it("copies the latest activation link", async () => {
    reissueInvite.mockResolvedValue({
      email: "newteacher@example.com",
      invite_token: "invite-token-reissued",
      activation_url: "http://localhost:3000/activate?token=invite-token-reissued",
      delivery_status: "sent",
    });
    writeText.mockResolvedValue(undefined);

    render(<UserManagementPanel data={data} />);

    fireEvent.click(screen.getByRole("tab", { name: "Pending invites" }));
    fireEvent.click(screen.getByRole("button", { name: "Resend email" }));

    await waitFor(() => expect(screen.getByLabelText("Latest activation link")).toHaveValue("http://localhost:3000/activate?token=invite-token-reissued"));

    fireEvent.click(screen.getByRole("button", { name: "Copy link" }));

    await waitFor(() => expect(writeText).toHaveBeenCalledWith("http://localhost:3000/activate?token=invite-token-reissued"));
    await waitFor(() => expect(screen.getByText("Copied")).toBeInTheDocument());
  });

  it("shows invite delivery failures from the API result", async () => {
    issueInvite.mockResolvedValue({
      email: "newteacher@example.com",
      invite_token: "invite-token",
      activation_url: "http://localhost:3000/activate?token=invite-token",
      delivery_status: "failed",
      delivery_error: "smtp offline",
    });

    render(<UserManagementPanel data={data} />);

    fireEvent.click(screen.getByRole("button", { name: "Invite user" }));
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "newteacher@example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Send invite" }));

    await waitFor(() => expect(screen.getAllByText("smtp offline")).toHaveLength(2));
  });

  it("does not crash when the API returns null lists", () => {
    const emptyData = {
      summary: {
        teachers: 0,
        parents: 0,
        pending_invites: 0,
        total_users: 0,
      },
      active_users: null,
      pending_invites: null,
    } as unknown as UserManagementView;

    render(
      <UserManagementPanel data={emptyData} />,
    );

    expect(screen.getByText("No active users match this search")).toBeInTheDocument();
  });
});
