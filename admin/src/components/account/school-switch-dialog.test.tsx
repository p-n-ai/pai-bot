import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import * as React from "react";
import { vi } from "vitest";
import { SchoolSwitchDialog } from "./school-switch-dialog";
import type { AuthSession, AuthUser, SchoolChoice } from "@/lib/api";

const {
  replaceMock,
  loadingMock,
  successMock,
  errorMock,
  switchSchoolMock,
  persistSessionMock,
  ensureQueryDataMock,
  setQueryDataMock,
  fetchDashboardProgressMock,
  fetchPreviewDashboardProgressMock,
} = vi.hoisted(() => ({
  replaceMock: vi.fn(),
  loadingMock: vi.fn(),
  successMock: vi.fn(),
  errorMock: vi.fn(),
  switchSchoolMock: vi.fn(),
  persistSessionMock: vi.fn(),
  ensureQueryDataMock: vi.fn(),
  setQueryDataMock: vi.fn(),
  fetchDashboardProgressMock: vi.fn(),
  fetchPreviewDashboardProgressMock: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    replace: replaceMock,
  }),
}));

vi.mock("sonner", () => ({
  toast: {
    loading: loadingMock,
    success: successMock,
    error: errorMock,
  },
}));

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({
    ensureQueryData: ensureQueryDataMock,
    setQueryData: setQueryDataMock,
  }),
  useMutation: ({ mutationFn, onMutate, onSuccess, onError }: {
    mutationFn: (variables: unknown) => Promise<unknown>;
    onMutate?: () => void;
    onSuccess?: (data: unknown) => void | Promise<void>;
    onError?: (error: unknown) => void;
  }) => ({
    isPending: false,
    mutate: (variables: unknown) => {
      onMutate?.();
      Promise.resolve()
        .then(() => mutationFn(variables))
        .then((result) => onSuccess?.(result))
        .catch((error) => onError?.(error));
    },
  }),
}));

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>();
  return {
    ...actual,
    switchSchool: switchSchoolMock,
    persistSession: persistSessionMock,
  };
});

vi.mock("@/lib/dashboard-progress-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/dashboard-progress-query")>();
  return {
    ...actual,
    fetchDashboardProgress: fetchDashboardProgressMock,
    fetchPreviewDashboardProgress: fetchPreviewDashboardProgressMock,
  };
});

const DialogContext = React.createContext<{ open: boolean; onOpenChange: (open: boolean) => void } | null>(null);
const SelectContext = React.createContext<{ value: string; onValueChange: (value: string) => void; triggerId?: string } | null>(null);

function findSelectItems(children: React.ReactNode): Array<{ value: string; label: string }> {
  return React.Children.toArray(children).flatMap((child) => {
    if (!React.isValidElement<Record<string, unknown>>(child)) {
      return [];
    }

    const next = findSelectItems(child.props.children as React.ReactNode);
    if (child.type === MockSelectItem) {
      return [{ value: String(child.props.value), label: String(child.props.children) }, ...next];
    }
    return next;
  });
}

function findTriggerId(children: React.ReactNode): string | undefined {
  for (const child of React.Children.toArray(children)) {
    if (!React.isValidElement<Record<string, unknown>>(child)) {
      continue;
    }
    if (typeof child.props.id === "string") {
      return child.props.id;
    }
    const nested = findTriggerId(child.props.children as React.ReactNode);
    if (nested) {
      return nested;
    }
  }
  return undefined;
}

function MockDialog({
  open,
  onOpenChange,
  children,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: React.ReactNode;
}) {
  return <DialogContext.Provider value={{ open, onOpenChange }}>{children}</DialogContext.Provider>;
}

function MockDialogTrigger({ render }: { render: React.ReactElement }) {
  const context = React.useContext(DialogContext);
  if (!context) return null;
  return React.cloneElement(render as React.ReactElement<{ onClick?: () => void }>, {
    onClick: () => context.onOpenChange(true),
  });
}

function MockDialogContent({ children }: { children: React.ReactNode }) {
  const context = React.useContext(DialogContext);
  if (!context?.open) return null;
  return <div>{children}</div>;
}

function MockDialogSection({ children }: { children: React.ReactNode }) {
  return <div>{children}</div>;
}

function MockSelect({
  value,
  onValueChange,
  children,
}: {
  value: string;
  onValueChange: (value: string) => void;
  children: React.ReactNode;
}) {
  const items = findSelectItems(children);
  const triggerId = findTriggerId(children);
  return (
    <SelectContext.Provider value={{ value, onValueChange, triggerId }}>
      <select
        aria-label="School"
        id={triggerId}
        value={value}
        onChange={(event) => onValueChange(event.target.value)}
      >
        {items.map((item) => (
          <option key={item.value} value={item.value}>
            {item.label}
          </option>
        ))}
      </select>
    </SelectContext.Provider>
  );
}

function MockSelectTrigger() {
  return null;
}

function MockSelectValue() {
  return null;
}

function MockSelectContent() {
  return null;
}

function MockSelectItem() {
  return null;
}

vi.mock("@/components/ui/dialog", () => ({
  Dialog: MockDialog,
  DialogTrigger: MockDialogTrigger,
  DialogContent: MockDialogContent,
  DialogHeader: MockDialogSection,
  DialogTitle: MockDialogSection,
  DialogDescription: MockDialogSection,
  DialogFooter: MockDialogSection,
}));

vi.mock("@/components/ui/select", () => ({
  Select: MockSelect,
  SelectTrigger: MockSelectTrigger,
  SelectValue: MockSelectValue,
  SelectContent: MockSelectContent,
  SelectItem: MockSelectItem,
}));

const currentUser: AuthUser = {
  user_id: "user-1",
  tenant_id: "tenant-a",
  tenant_slug: "school-a",
  tenant_name: "School A",
  role: "teacher",
  name: "Teacher A",
  email: "teacher@example.com",
};

const schoolChoices: SchoolChoice[] = [
  {
    tenant_id: "tenant-a",
    tenant_slug: "school-a",
    tenant_name: "School A",
  },
  {
    tenant_id: "tenant-b",
    tenant_slug: "school-b",
    tenant_name: "School B",
  },
];

const switchedSession: AuthSession = {
  expires_at: "2026-04-07T00:00:00Z",
  user: {
    ...currentUser,
    tenant_id: "tenant-b",
    tenant_slug: "school-b",
    tenant_name: "School B",
  },
  tenant_choices: schoolChoices,
};

describe("SchoolSwitchDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    fetchDashboardProgressMock.mockResolvedValue({ source: "live", progress: { students: [], topic_ids: [] } });
    fetchPreviewDashboardProgressMock.mockResolvedValue({ source: "preview", progress: { students: [], topic_ids: [] } });
    ensureQueryDataMock.mockImplementation(async ({ queryFn }: { queryFn: () => Promise<unknown> }) => queryFn());
  });

  it("does not render when the account cannot switch schools", () => {
    render(<SchoolSwitchDialog currentUser={currentUser} schoolChoices={[schoolChoices[0]]} />);

    expect(screen.queryByRole("button", { name: "Switch school" })).not.toBeInTheDocument();
  });

  it("requires a different school and password before enabling submission", () => {
    render(
      <SchoolSwitchDialog
        currentUser={currentUser}
        schoolChoices={schoolChoices}
        triggerLabel="Change school"
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Change school" }));

    const submit = screen.getByRole("button", { name: "Switch school" });
    expect(submit).toBeDisabled();

    fireEvent.change(screen.getByLabelText("School"), { target: { value: "tenant-b" } });
    expect(submit).toBeDisabled();

    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "secret-pass" } });
    expect(submit).toBeEnabled();
  });

  it("switches school, persists the session, and prefetches the tenant-keyed dashboard query", async () => {
    switchSchoolMock.mockResolvedValue(switchedSession);

    render(
      <SchoolSwitchDialog
        currentUser={currentUser}
        schoolChoices={schoolChoices}
        triggerLabel="Change school"
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Change school" }));
    fireEvent.change(screen.getByLabelText("School"), { target: { value: "tenant-b" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "secret-pass" } });
    fireEvent.click(screen.getByRole("button", { name: "Switch school" }));

    await waitFor(() => expect(switchSchoolMock).toHaveBeenCalledWith("tenant-b", "secret-pass"));
    await waitFor(() => expect(persistSessionMock).toHaveBeenCalledWith(switchedSession));
    await waitFor(() =>
      expect(ensureQueryDataMock).toHaveBeenCalledWith(
        expect.objectContaining({
          queryKey: ["dashboard-progress", "tenant-b"],
        }),
      ),
    );
    await waitFor(() => expect(fetchDashboardProgressMock).toHaveBeenCalledWith("tenant-b"));
    await waitFor(() => expect(successMock).toHaveBeenCalledWith("School changed to School B.", { id: "school-switch" }));
    await waitFor(() => expect(replaceMock).toHaveBeenCalledWith("/dashboard"));
  });

  it("falls back to preview dashboard data when tenant prefetch fails", async () => {
    switchSchoolMock.mockResolvedValue(switchedSession);
    ensureQueryDataMock.mockRejectedValue(new Error("live fetch failed"));

    render(
      <SchoolSwitchDialog
        currentUser={currentUser}
        schoolChoices={schoolChoices}
        triggerLabel="Change school"
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Change school" }));
    fireEvent.change(screen.getByLabelText("School"), { target: { value: "tenant-b" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "secret-pass" } });
    fireEvent.click(screen.getByRole("button", { name: "Switch school" }));

    await waitFor(() => expect(fetchPreviewDashboardProgressMock).toHaveBeenCalled());
    await waitFor(() =>
      expect(setQueryDataMock).toHaveBeenCalledWith(
        ["dashboard-progress", "tenant-b"],
        expect.objectContaining({ source: "preview" }),
      ),
    );
  });

  it("surfaces backend switch failures without mutating session state", async () => {
    switchSchoolMock.mockRejectedValue(new Error("Password check failed"));

    render(
      <SchoolSwitchDialog
        currentUser={currentUser}
        schoolChoices={schoolChoices}
        triggerLabel="Change school"
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Change school" }));
    fireEvent.change(screen.getByLabelText("School"), { target: { value: "tenant-b" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "secret-pass" } });
    fireEvent.click(screen.getByRole("button", { name: "Switch school" }));

    await waitFor(() => expect(errorMock).toHaveBeenCalledWith("Password check failed", { id: "school-switch" }));
    expect(persistSessionMock).not.toHaveBeenCalled();
    expect(replaceMock).not.toHaveBeenCalled();
    expect(screen.getByText("Password check failed")).toBeInTheDocument();
  });
});
