import { fireEvent, render, screen } from "@testing-library/react";
import { vi } from "vitest";
import { InviteIssueForm } from "@/components/invite-issue-form";

describe("InviteIssueForm", () => {
  it("renders email, role, and action state", () => {
    render(
      <InviteIssueForm
        email=""
        role="teacher"
        error=""
        inviteLink=""
        deliveryStatus="pending"
        copyFeedback=""
        isPending={false}
        onEmailChange={() => {}}
        onRoleChange={() => {}}
        onCopyLink={() => {}}
        onSubmit={(event) => event.preventDefault()}
      />,
    );

    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.getByLabelText("Role")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send invite" })).toBeInTheDocument();
  });

  it("shows the generated activation link after a successful invite", () => {
    const onCopyLink = vi.fn();

    render(
      <InviteIssueForm
        email="teacher@example.com"
        role="teacher"
        error=""
        inviteLink="http://localhost:3000/activate?token=invite-token"
        deliveryStatus="sent"
        copyFeedback=""
        isPending={false}
        onEmailChange={() => {}}
        onRoleChange={() => {}}
        onCopyLink={onCopyLink}
        onSubmit={(event) => event.preventDefault()}
      />,
    );

    expect(screen.getByDisplayValue("http://localhost:3000/activate?token=invite-token")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Copy link" }));
    expect(onCopyLink).toHaveBeenCalledTimes(1);
  });

  it("submits the invite request", () => {
    let submitted = false;

    render(
      <InviteIssueForm
        email="teacher@example.com"
        role="teacher"
        error=""
        inviteLink=""
        deliveryStatus="pending"
        copyFeedback=""
        isPending={false}
        onEmailChange={() => {}}
        onRoleChange={() => {}}
        onCopyLink={() => {}}
        onSubmit={(event) => {
          event.preventDefault();
          submitted = true;
        }}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Send invite" }));

    expect(submitted).toBe(true);
  });

  it("shows a delivery failure note when email delivery fails", () => {
    render(
      <InviteIssueForm
        email="teacher@example.com"
        role="teacher"
        error=""
        inviteLink="http://localhost:3000/activate?token=invite-token"
        deliveryStatus="failed"
        deliveryError="smtp offline"
        copyFeedback=""
        isPending={false}
        onEmailChange={() => {}}
        onRoleChange={() => {}}
        onCopyLink={() => {}}
        onSubmit={(event) => event.preventDefault()}
      />,
    );

    expect(screen.getByText("smtp offline")).toBeInTheDocument();
  });
});
