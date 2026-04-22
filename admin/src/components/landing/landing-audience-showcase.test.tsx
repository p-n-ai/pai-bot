import { fireEvent, render, screen } from "@testing-library/react";
import { LandingAudienceShowcase } from "./landing-audience-showcase";

describe("LandingAudienceShowcase", () => {
  it("switches between student and teacher landing views", () => {
    render(
      <LandingAudienceShowcase
        primaryHref="/login"
        primaryActionLabel="Sign in"
      />,
    );

    expect(screen.getByRole("tab", { name: /students/i })).toHaveAttribute("aria-selected", "true");
    expect(document.querySelector("#student-panel")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: /teachers/i }));

    expect(screen.getByRole("tab", { name: /teachers/i })).toHaveAttribute("aria-selected", "true");
    expect(document.querySelector("#teacher-panel")).toBeInTheDocument();
  });
});
