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
        name: /learn math in chat\./i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /stay with the problem\./i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: /teachers/i }));

    expect(
      await screen.findByRole("heading", {
        name: /see the next follow-up\./i,
      }),
    ).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: /follow-up, in order\./i })).toBeInTheDocument();
  });
});
