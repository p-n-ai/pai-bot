import { fireEvent, render, screen } from "@testing-library/react";
import { LoginFormCard } from "@/components/login-form-card";

describe("LoginFormCard", () => {
  it("renders the default sign-in form state", () => {
    render(
      <LoginFormCard
        email=""
        password=""
        error=""
        isPending={false}
        onEmailChange={() => {}}
        onPasswordChange={() => {}}
        onSubmit={(event) => event.preventDefault()}
      />,
    );

    expect(screen.getByRole("heading", { name: "Sign in" })).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.queryByText("School")).not.toBeInTheDocument();
  });

  it("does not render a school selector even when tenant choices exist", () => {
    render(
      <LoginFormCard
        email="teacher@example.com"
        password="secret"
        error="Choose the correct school to continue."
        isPending={false}
        onEmailChange={() => {}}
        onPasswordChange={() => {}}
        onSubmit={(event) => event.preventDefault()}
      />,
    );

    expect(screen.getByText("Choose the correct school to continue.")).toBeInTheDocument();
    expect(screen.queryByLabelText("School")).not.toBeInTheDocument();
  });

  it("submits the form through the shared action button", () => {
    let submitted = false;

    render(
      <LoginFormCard
        email="teacher@example.com"
        password="secret"
        error=""
        isPending={false}
        onEmailChange={() => {}}
        onPasswordChange={() => {}}
        onSubmit={(event) => {
          event.preventDefault();
          submitted = true;
        }}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));

    expect(submitted).toBe(true);
  });
});
