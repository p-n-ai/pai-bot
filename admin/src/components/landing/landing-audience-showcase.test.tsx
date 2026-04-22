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
        name: /unstuck in chat\./i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /hint\. check\. continue\./i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: /teachers/i }));

    expect(
      await screen.findByRole("heading", {
        name: /fix the right topic\./i,
      }),
    ).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: /daily follow-up list\./i })).toBeInTheDocument();
  });
});
