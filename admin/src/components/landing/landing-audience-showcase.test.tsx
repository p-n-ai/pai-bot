import { fireEvent, render, screen } from "@testing-library/react";
import { LandingAudienceShowcase } from "./landing-audience-showcase";

describe("LandingAudienceShowcase", () => {
  it("switches between student and teacher landing views", async () => {
    render(
      <LandingAudienceShowcase
        primaryHref="/login"
        primaryActionLabel="Sign in"
      />,
    );

    expect(
      screen.getByRole("heading", {
        name: /students get unstuck in chat\./i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /teach the next step, then check it\./i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: /teachers/i }));

    expect(
      await screen.findByRole("heading", {
        name: /teachers see what to fix first\./i,
      }),
    ).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: /a daily list, not another report\./i })).toBeInTheDocument();
  });
});
