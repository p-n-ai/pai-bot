import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { UserManagementPanel } from "@/components/user-management-panel";

const { issueInvite } = vi.hoisted(() => ({
  issueInvite: vi.fn(),
}));

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>();
  return {
    ...actual,
    issueInvite,
  };
});

const data = {
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
      expires_at: "2026-04-13T10:00:00Z",
      created_at: "2026-04-06T10:00:00Z",
      invited_by: "Admin User",
    },
  ],
} as const;

describe("UserManagementPanel", () => {
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
      invite_token: "invite-token",
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
    expect(screen.getByDisplayValue("http://localhost:3000/activate?token=invite-token")).toBeInTheDocument();
  });

  it("does not crash when the API returns null lists", () => {
    render(
      <UserManagementPanel
        data={
          {
            summary: {
              teachers: 0,
              parents: 0,
              pending_invites: 0,
              total_users: 0,
            },
            active_users: null,
            pending_invites: null,
          } as any
        }
      />,
    );

    expect(screen.getByText("No active users match this search")).toBeInTheDocument();
  });
});
