import { render, screen } from "@testing-library/react";
import { TopicProgressRow } from "@/components/topic-progress-row";

describe("TopicProgressRow", () => {
  it("renders topic title, status, and progress", () => {
    render(<TopicProgressRow title="Linear Equations" status="Live" progress={0.76} />);

    expect(screen.getByText("Linear Equations")).toBeInTheDocument();
    expect(screen.getByText("Live")).toBeInTheDocument();
    expect(screen.getByText("76%")).toBeInTheDocument();
  });
});
