import { By, until } from 'selenium-webdriver'
import { isOverlapping, setupDrivers } from '../TestHelpers'


describe("Course utility elements should not overlap", () => {

    const overlapTests: { width: number, height: number, want: boolean }[] = [
        // Resolutions to test
        { width: 1920, height: 1080, want: false }, // Desktop
        { width: 360, height: 740, want: false },   // More common phones
        { width: 360, height: 640, want: false },   // Older phones
        { width: 412, height: 914, want: false }    // Bigger phones
    ]

    const drivers = setupDrivers("/course/1")

    drivers.forEach(driver => {
        overlapTests.forEach(test => {
            it(`Should not overlap on split screen ${test.width}x${test.height}`, async () => {
                await driver.manage().window().setRect({ width: test.width, height: test.height })
                if (await driver.findElement(By.className("closeButton")).isDisplayed()) {
                    driver.findElement(By.className("closeButton")).click()
                }
                const switchStudent = driver.findElement(By.className("clickable"))
                await driver.wait(until.elementIsVisible(switchStudent), 100)
                await switchStudent.click()

                const labs = await driver.findElement(By.className("col-md-9"))
                await driver.wait(until.elementIsVisible(labs), 5000)

                const utility = await driver.findElement(By.className("list-group width-resize"))
                await driver.wait(until.elementIsVisible(utility), 5000)
                const rect = await utility.getRect()
                const rect2 = await labs.getRect()

                const overlap = isOverlapping(rect, rect2)

                expect(overlap).toBe(test.want)

                // Refresh the page to reset the state
                await driver.navigate().refresh()
            }, 50000)
        })
    })
})
