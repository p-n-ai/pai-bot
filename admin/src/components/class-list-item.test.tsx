import { fireEvent, render, screen } from "@testing-library/react";
import { ClassListItem } from "@/components/class-list-item";

describe("ClassListItem", () => {
  it("renders class metadata", () => {
    render(
      <ClassListItem
        name="Form 1 Algebra A"
        syllabus="KSSM Form 1"
        joinCode="ALG-F1A"
        summary="Early algebra cohort."
        active={false}
        onClick={() => {}}
      />,
    );

    expect(screen.getByRole("button", { name: /form 1 algebra a/i })).toBeInTheDocument();
    expect(screen.getByText("ALG-F1A")).toBeInTheDocument();
  });

  it("calls onClick when selected", () => {
    let clicked = false;

    render(
      <ClassListItem
        name="Form 2 Algebra B"
        syllabus="KSSM Form 2"
        joinCode="ALG-F2B"
        summary="Mixed-ability class."
        active
        onClick={() => {
          clicked = true;
        }}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /form 2 algebra b/i }));

    expect(clicked).toBe(true);
  });
});
