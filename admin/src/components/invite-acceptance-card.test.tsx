import { fireEvent, render, screen } from "@testing-library/react";
import { InviteAcceptanceCard } from "@/components/invite-acceptance-card";

describe("InviteAcceptanceCard", () => {
  it("renders the activation form when an invite token is present", () => {
    render(
      <InviteAcceptanceCard
        token="invite-token"
        name=""
        password=""
        error=""
        isPending={false}
        onNameChange={() => {}}
        onPasswordChange={() => {}}
        onSubmit={(event) => event.preventDefault()}
      />,
    );

    expect(screen.getByRole("heading", { name: "Accept your invite" })).toBeInTheDocument();
    expect(screen.getByLabelText("Full name")).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
  });

  it("shows the missing-token state and disables activation", () => {
    render(
      <InviteAcceptanceCard
        token=""
        name=""
        password=""
        error="Invite token missing."
        isPending={false}
        onNameChange={() => {}}
        onPasswordChange={() => {}}
        onSubmit={(event) => event.preventDefault()}
      />,
    );

    expect(screen.getByText("Invite token missing.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Accept invite" })).toBeDisabled();
  });

  it("submits the activation form", () => {
    let submitted = false;

    render(
      <InviteAcceptanceCard
        token="invite-token"
        name="Parent One"
        password="strong-pass-1"
        error=""
        isPending={false}
        onNameChange={() => {}}
        onPasswordChange={() => {}}
        onSubmit={(event) => {
          event.preventDefault();
          submitted = true;
        }}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Accept invite" }));

    expect(submitted).toBe(true);
  });
});
