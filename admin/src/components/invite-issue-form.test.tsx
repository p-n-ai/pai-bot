import { fireEvent, render, screen } from "@testing-library/react";
import { InviteIssueForm } from "@/components/invite-issue-form";

describe("InviteIssueForm", () => {
  it("renders email, role, and action state", () => {
    render(
      <InviteIssueForm
        email=""
        role="teacher"
        error=""
        inviteLink=""
        isPending={false}
        onEmailChange={() => {}}
        onRoleChange={() => {}}
        onSubmit={(event) => event.preventDefault()}
      />,
    );

    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.getByLabelText("Role")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send invite" })).toBeInTheDocument();
  });

  it("shows the generated activation link after a successful invite", () => {
    render(
      <InviteIssueForm
        email="teacher@example.com"
        role="teacher"
        error=""
        inviteLink="http://localhost:3000/activate?token=invite-token"
        isPending={false}
        onEmailChange={() => {}}
        onRoleChange={() => {}}
        onSubmit={(event) => event.preventDefault()}
      />,
    );

    expect(screen.getByDisplayValue("http://localhost:3000/activate?token=invite-token")).toBeInTheDocument();
  });

  it("submits the invite request", () => {
    let submitted = false;

    render(
      <InviteIssueForm
        email="teacher@example.com"
        role="teacher"
        error=""
        inviteLink=""
        isPending={false}
        onEmailChange={() => {}}
        onRoleChange={() => {}}
        onSubmit={(event) => {
          event.preventDefault();
          submitted = true;
        }}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Send invite" }));

    expect(submitted).toBe(true);
  });
});
