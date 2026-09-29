package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

// Everything that makes the screen look like a Spring Boot log. Pure string generation.
const (
	pid      = 48213
	port     = 8080
	appName  = "DemoApplication"
	lineGap  = 3 * time.Millisecond // fake ms between consecutive log lines
	tsLayout = "2006-01-02 15:04:05.000"
	// Spring Boot 2.x console pattern with a 26-col logger column (Spring uses 40) so a
	// board row fits in 120 cols. Long boot loggers overflow the column, as in real logs.
	lineFmt = "%s %5s %d --- [%15s] %-26s : %s"
)

const banner = `
  .   ____          _            __ _ _
 /\\ / ___'_ __ _ _(_)_ __  __ _ \ \ \ \
( ( )\___ | '_ | '_| | '_ \/ _` + "`" + ` | \ \ \ \
 \\/  ___)| |_)| | | | | || (_| |  ) ) ) )
  '  |____| .__|_| |_|_| |_\__, | / / / /
 =========|_|==============|___/=/_/_/_/

 :: Spring Boot ::                (v2.7.18)
`

var (
	// Thread names as logback shows them: right-aligned and left-truncated to 15 chars.
	execThreads = []string{
		"nio-8080-exec-1", "nio-8080-exec-2", "nio-8080-exec-3", "nio-8080-exec-4",
		"nio-8080-exec-5", "nio-8080-exec-6", "nio-8080-exec-7", "nio-8080-exec-8",
	}
	boardLoggers = []string{
		"c.a.d.service.OrderService", "c.a.d.api.OrderController", "o.s.w.s.DispatcherServlet",
		"c.a.d.repo.OrderRepository", "o.h.SQL", "c.a.d.service.PriceService",
	}
	endpoints = []string{
		`GET "/api/orders/1042"`, `GET "/api/orders?page=3&size=50"`, `POST "/api/orders"`,
		`GET "/actuator/health"`, `PUT "/api/orders/1042/status"`, `GET "/api/customers/88/orders"`,
	}
)

func logLine(t time.Time, level, thread, logger, msg string) string {
	return fmt.Sprintf(lineFmt, t.Format(tsLayout), level, pid, thread, logger, msg)
}

// boardLine dresses a board row as a DEBUG line. Thread and logger are chosen by row
// index so the columns do not flicker between ticks.
func boardLine(now time.Time, y int, row string) string {
	return logLine(now.Add(time.Duration(y)*lineGap), "DEBUG",
		execThreads[y%len(execThreads)], boardLoggers[y%len(boardLoggers)], row)
}

func bootLines(t time.Time) []string {
	lines := strings.Split(strings.Trim(banner, "\n"), "\n")
	main := func(logger, msg string) {
		t = t.Add(lineGap)
		lines = append(lines, logLine(t, "INFO", "main", logger, msg))
	}
	main("c.a.demo."+appName, fmt.Sprintf("Starting %s v0.0.1-SNAPSHOT using Java 17.0.9 on prod-api-01 with PID %d (/app/app.jar started by app in /app)", appName, pid))
	main("c.a.demo."+appName, `The following 1 profile is active: "prod"`)
	main("o.s.d.r.c.RepositoryConfigurationDelegate", "Bootstrapping Spring Data JPA repositories in DEFAULT mode.")
	main("o.s.d.r.c.RepositoryConfigurationDelegate", "Finished Spring Data repository scanning in 41 ms. Found 3 JPA repository interfaces.")
	main("o.s.b.w.e.tomcat.TomcatWebServer", fmt.Sprintf("Tomcat initialized with port %d (http)", port))
	main("o.a.catalina.core.StandardService", `Starting service [Tomcat]`)
	main("o.a.catalina.core.StandardEngine", `Starting Servlet engine: [Apache Tomcat/9.0.83]`)
	main("o.s.w.c.ContextLoader", "Root WebApplicationContext: initialization completed in 1187 ms")
	main("o.h.jpa.internal.util.LogHelper", "HHH000204: Processing PersistenceUnitInfo [name: default]")
	main("org.hibernate.Version", "HHH000412: Hibernate ORM core version 5.6.15.Final")
	main("c.z.hikari.HikariDataSource", "HikariPool-1 - Starting...")
	main("c.z.hikari.pool.HikariPool", "HikariPool-1 - Added connection org.postgresql.jdbc.PgConnection@6b3e12f0")
	main("c.z.hikari.HikariDataSource", "HikariPool-1 - Start completed.")
	main("o.s.b.a.e.web.EndpointLinksResolver", "Exposing 3 endpoints beneath base path '/actuator'")
	main("o.s.b.w.e.tomcat.TomcatWebServer", fmt.Sprintf("Tomcat started on port %d (http) with context path '/'", port))
	main("c.a.demo."+appName, fmt.Sprintf("Started %s in 3.214 seconds (process running for 3.6)", appName))
	return lines
}

func trafficLines(t time.Time) []string {
	switch rand.IntN(4) {
	case 0:
		return []string{logLine(t, "INFO", "scheduling-1", "c.a.demo.jobs.ReconcileJob",
			fmt.Sprintf("Reconciled %d orders in %d ms", rand.IntN(400), 20+rand.IntN(300)))}
	case 1:
		return []string{logLine(t, "DEBUG", "l-1 housekeeper", "c.z.hikari.pool.HikariPool",
			fmt.Sprintf("HikariPool-1 - Pool stats (total=10, active=%d, idle=%d, waiting=0)", rand.IntN(4), 6+rand.IntN(4)))}
	default:
		thread := execThreads[rand.IntN(len(execThreads))]
		return []string{
			logLine(t, "DEBUG", thread, "o.s.w.s.DispatcherServlet", endpoints[rand.IntN(len(endpoints))]+", parameters={}"),
			logLine(t.Add(lineGap), "DEBUG", thread, "o.s.w.s.DispatcherServlet", "Completed 200 OK"),
		}
	}
}

// crashLines is the game-over screen: a stack trace with the score hidden as a latency.
func crashLines(t time.Time, cause string, score int) []string {
	thread := execThreads[rand.IntN(len(execThreads))]
	lines := []string{
		logLine(t, "ERROR", thread, "o.a.c.c.C.[.[.[/].[dispatcherServlet]",
			"Servlet.service() for servlet [dispatcherServlet] threw exception"),
		"java.lang." + cause + ": Cannot invoke \"com.acme.demo.model.Order.getItems()\" because \"order\" is null",
		"        at com.acme.demo.service.OrderService.total(OrderService.java:142)",
		"        at com.acme.demo.service.OrderService.lambda$reprice$3(OrderService.java:97)",
		"        at java.base/java.util.stream.ReferencePipeline$3$1.accept(ReferencePipeline.java:197)",
		"        at com.acme.demo.api.OrderController.reprice(OrderController.java:61)",
		"        at java.base/jdk.internal.reflect.DirectMethodHandleAccessor.invoke(DirectMethodHandleAccessor.java:103)",
		"        at org.springframework.web.method.support.InvocableHandlerMethod.doInvoke(InvocableHandlerMethod.java:257)",
		"        at org.springframework.web.servlet.DispatcherServlet.doDispatch(DispatcherServlet.java:1089)",
		"        at org.apache.tomcat.util.threads.TaskThread$WrappingRunnable.run(TaskThread.java:63)",
		"        at java.base/java.lang.Thread.run(Thread.java:1583)",
		logLine(t.Add(lineGap), "INFO", thread, "o.s.w.s.DispatcherServlet", fmt.Sprintf("Completed 200 OK (%d ms)", score)),
	}
	return lines
}

func shutdownLines(t time.Time) []string {
	return []string{
		logLine(t, "INFO", "ionShutdownHook", "o.s.b.w.e.tomcat.GracefulShutdown", "Commencing graceful shutdown. Waiting for active requests to complete"),
		logLine(t.Add(lineGap), "INFO", "ionShutdownHook", "o.s.s.c.ThreadPoolTaskExecutor", "Shutting down ExecutorService 'applicationTaskExecutor'"),
		logLine(t.Add(2*lineGap), "INFO", "ionShutdownHook", "c.z.hikari.HikariDataSource", "HikariPool-1 - Shutdown initiated..."),
		logLine(t.Add(3*lineGap), "INFO", "ionShutdownHook", "c.z.hikari.HikariDataSource", "HikariPool-1 - Shutdown completed."),
	}
}
