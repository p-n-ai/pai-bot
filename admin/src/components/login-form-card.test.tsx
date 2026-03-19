import { fireEvent, render, screen } from "@testing-library/react";
import { LoginFormCard } from "@/components/login-form-card";

describe("LoginFormCard", () => {
  it("renders the default sign-in form state", () => {
    render(
      <LoginFormCard
        email=""
        password=""
        tenantID=""
        tenantChoices={[]}
        error=""
        isPending={false}
        onEmailChange={() => {}}
        onPasswordChange={() => {}}
        onTenantChange={() => {}}
        onSubmit={(event) => event.preventDefault()}
      />,
    );

    expect(screen.getByRole("heading", { name: "Sign in" })).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.queryByText("School")).not.toBeInTheDocument();
  });

  it("shows tenant selection and error state when tenant context is required", () => {
    render(
      <LoginFormCard
        email="teacher@example.com"
        password="secret"
        tenantID="tenant-1"
        tenantChoices={[
          { tenant_id: "tenant-1", tenant_name: "Sekolah Pandai", tenant_slug: "pandai" },
          { tenant_id: "tenant-2", tenant_name: "Sekolah Beta", tenant_slug: "beta" },
        ]}
        error="Choose the correct school to continue."
        isPending={false}
        onEmailChange={() => {}}
        onPasswordChange={() => {}}
        onTenantChange={() => {}}
        onSubmit={(event) => event.preventDefault()}
      />,
    );

    expect(screen.getByText("School")).toBeInTheDocument();
    expect(screen.getByText("Choose the correct school to continue.")).toBeInTheDocument();
    expect(screen.getByLabelText("School")).toBeInTheDocument();
  });

  it("submits the form through the shared action button", () => {
    let submitted = false;

    render(
      <LoginFormCard
        email="teacher@example.com"
        password="secret"
        tenantID=""
        tenantChoices={[]}
        error=""
        isPending={false}
        onEmailChange={() => {}}
        onPasswordChange={() => {}}
        onTenantChange={() => {}}
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
