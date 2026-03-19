import { render, screen } from "@testing-library/react";
import { Input } from "@/components/ui/input";
import { FormField } from "@/components/form-field";

describe("FormField", () => {
  it("renders label, description, and child control", () => {
    render(
      <FormField label="Email" htmlFor="email" description="Use your school email.">
        <Input id="email" />
      </FormField>,
    );

    expect(screen.getByText("Email")).toBeInTheDocument();
    expect(screen.getByText("Use your school email.")).toBeInTheDocument();
    expect(screen.getByRole("textbox")).toBeInTheDocument();
  });

  it("renders error text when provided", () => {
    render(
      <FormField label="Password" htmlFor="password" error="Password is required.">
        <Input id="password" type="password" />
      </FormField>,
    );

    expect(screen.getByText("Password is required.")).toBeInTheDocument();
  });
});
