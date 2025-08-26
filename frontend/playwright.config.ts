import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/e2e",
  /* Run tests in files in parallel */
  fullyParallel: true,
  /* Fail the build on CI if you accidentally left test.only in the source code. */
  forbidOnly: !!process.env.CI,
  /* Retry on CI only */
  retries: process.env.CI ? 2 : 0,
  /* Optimized worker configuration for parallel execution */
  workers: process.env.CI ? 2 : "50%",
  /* Enhanced reporting with multiple formats */
  reporter: [
    ["html", { outputFolder: "playwright-report" }],
    ["json", { outputFile: "test-results/results.json" }],
    ["junit", { outputFile: "test-results/junit.xml" }],
    ["list"],
  ],
  /* Test output directory */
  outputDir: "test-results/",
  /* Shared settings for all the projects below. See https://playwright.dev/docs/api/class-testoptions. */
  use: {
    /* Base URL to use in actions like `await page.goto('/')`. */
    baseURL: process.env.CI ? "http://localhost:4173" : "http://localhost:3000",

    /* Collect trace when retrying the failed test. See https://playwright.dev/docs/trace-viewer */
    trace: "on-first-retry",

    /* Enhanced debugging and monitoring */
    screenshot: "only-on-failure",
    video: "retain-on-failure",

    /* Optimized timeouts for performance */
    actionTimeout: 15000,
    navigationTimeout: 30000,

    /* Browser context options for stability */
    viewport: { width: 1280, height: 720 },
    ignoreHTTPSErrors: true,

    /* Optimized for parallel execution */
    launchOptions: {
      args: [
        "--disable-extensions",
        "--disable-gpu",
        "--disable-web-security",
        "--allow-running-insecure-content",
        "--disable-features=TranslateUI,BlinkGenPropertyTrees",
      ],
    },
  },

  /* Configure projects for major browsers */
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },

    {
      name: "webkit",
      use: { ...devices["Desktop Safari"] },
    },

    /* Test against mobile viewports. */
    {
      name: "Mobile Chrome",
      use: { ...devices["Pixel 5"] },
    },
    {
      name: "Mobile Safari",
      use: { ...devices["iPhone 12"] },
    },

    /* Test against branded browsers. */
    // {
    //   name: 'Microsoft Edge',
    //   use: { ...devices['Desktop Edge'], channel: 'msedge' },
    // },
    // {
    //   name: 'Google Chrome',
    //   use: { ...devices['Desktop Chrome'], channel: 'chrome' },
    // },
  ],

  /* Skip webServer config - assume Docker containers are already running */
});
