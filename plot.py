import pandas as pd
import matplotlib.pyplot as plt
import matplotlib as mpl
import numpy as np

def plot_mem(program):
    goroutines = [1, 5, 10]
    for idx, g in enumerate(goroutines):
        df_gc = pd.read_csv("results/" + program + "/" + str(g) + "-GC-mem.csv")
        df_rbmm = pd.read_csv("results/" + program + "/" + str(g) + "-RBMM-mem.csv")

        df_rbmm = df_rbmm.drop_duplicates(subset="Time", keep="first")
        df_gc = df_gc.drop_duplicates(subset="Time", keep="first")

        metrics = ["M_C", "ExtFrag", "IntFrag"]
        metric_labels = ["Memory Consumption", "External Fragmentation", "Internal Fragmentation"]
        colors = ["purple", "orange"]  # Different colors for different metrics
        linestyles = ["-", "--"]  # Solid for RBMM, Dashed for GC

        common_time = np.union1d(df_rbmm["Time"], df_gc["Time"])

        df_rbmm_interp = df_rbmm.set_index("Time").reindex(common_time).interpolate().reset_index()
        df_gc_interp = df_gc.set_index("Time").reindex(common_time).interpolate().reset_index()

        df_rbmm_interp.rename(columns={"index": "Time"}, inplace=True)
        df_gc_interp.rename(columns={"index": "Time"}, inplace=True)

        fig, axes = plt.subplots(nrows=3, ncols=1, figsize=(10, 12))
        for i, metric in enumerate(metrics):
            ax = axes[i]

            # Plot RBMM
            ax.plot(df_rbmm_interp["Time"], df_rbmm_interp[metric],
                    color=colors[0], linestyle=linestyles[0], label="RBMM")


            # Plot GC
            ax.plot(df_gc_interp["Time"], df_gc_interp[metric],
                    color=colors[1], linestyle=linestyles[1], label="GC")


            # Formatting
            ax.set_ylabel(metric_labels[i] + " (MB)")
            ax.set_xlabel("Time (ms)")
            ax.grid(True)
            legend = ax.legend(loc="upper left")
            legend.get_frame().set_edgecolor("black")  # Set border color to black
            legend.get_frame().set_boxstyle("Square")  # No rounded corners

        # Set a common title
        plt.suptitle("Memory efficiency: " + program + "-" + str(g))

        plt.tight_layout(rect=[0, 0, 1, 0.96])
        plt.show(block=False)

def plot_sys(program):
    df_gc = pd.read_csv("results/" + program + "/" + "GC-sys.csv")
    df_rbmm = pd.read_csv("results/" + program + "/" + "RBMM-sys.csv")

    df_rbmm = df_rbmm.drop_duplicates(subset="G", keep="first")
    df_gc = df_gc.drop_duplicates(subset="G", keep="first")

    metrics = ["T_C", "T_L", "T_A", "T_D", "Theta"]
    error_metrics = ["T_C_ERR", "T_L_ERR", "T_A_ERR", "T_D_ERR", "Theta_ERR"]
    metric_labels = ["Computation time (ms)", "Latency (ms)", "Allocation time (ms)", "Deallocation time (ms)", "Throughput (op/ms)"]
    colors = ["hotpink", "blue"]  # Different colors for different metrics
    linestyles = ["-", "--"]  # Solid for RBMM, Dashed for GC

    common_goroutines = np.union1d(df_rbmm["G"], df_gc["G"])

    df_rbmm_interp = df_rbmm.set_index("G").reindex(common_goroutines).interpolate().reset_index()
    df_gc_interp = df_gc.set_index("G").reindex(common_goroutines).interpolate().reset_index()

    df_rbmm_interp.rename(columns={"index": "G"}, inplace=True)
    df_gc_interp.rename(columns={"index": "G"}, inplace=True)

    fig, axes = plt.subplots(nrows=5, ncols=1, figsize=(10, 12))
    for i, metric in enumerate(metrics):
        ax = axes[i]
        err_metric = error_metrics[i]
        if metric == "Theta":
            x = np.arange(len(df_rbmm_interp["G"]))
            bar_width = 0.1
            # Plot RBMM
            ax.bar(x - bar_width / 2, df_rbmm_interp[metric],
                    color=colors[0], label="RBMM", width = bar_width)

            ax.errorbar(x - bar_width / 2, df_rbmm_interp[metric],
                        yerr=df_rbmm_interp[err_metric], fmt="none", color="black",
                        capsize=3, elinewidth=1)

            # Plot GC
            ax.bar(x + bar_width / 2, df_gc_interp[metric],
                    color=colors[1], label="GC", width = bar_width)

            ax.errorbar(x + bar_width / 2, df_gc_interp[metric],
                        yerr=df_gc_interp[err_metric], fmt="none", color="black",
                        capsize=3, elinewidth=1)

            ax.set_ylabel(metric_labels[i])
            ax.set_xticks(x)
            ax.set_xticklabels(df_rbmm_interp["G"])
        else:
            # Plot RBMM
            ax.plot(df_rbmm_interp["G"], df_rbmm_interp[metric],
                        linestyle=linestyles[0], color=colors[0], label="RBMM", marker="o")
            ax.errorbar(df_rbmm_interp["G"], df_rbmm_interp[metric],
                        yerr=df_rbmm_interp[err_metric], fmt="o", color=colors[0], markerfacecolor=colors[0],
                        markeredgecolor=colors[0], label="", capsize=3, elinewidth=1)

            # Plot GC
            ax.plot(df_gc_interp["G"], df_gc_interp[metric],
                        linestyle=linestyles[1], color=colors[1], label="GC", marker="s")
            ax.errorbar(df_gc_interp["G"], df_gc_interp[metric],
                        yerr=df_gc_interp[err_metric], fmt="s", color=colors[1], markerfacecolor=colors[1],
                        markeredgecolor=colors[1], label="", capsize=3, elinewidth=1)

            ax.set_ylabel(metric_labels[i])
            ax.grid(True)

        ax.set_xlabel("Goroutines")
        legend = ax.legend(loc="upper left")
        legend.get_frame().set_edgecolor("black")  # Set border color to black
        legend.get_frame().set_boxstyle("Square")  # No rounded corners

    # Set a common title
    plt.suptitle("System performance: " + program)
    plt.tight_layout(rect=[0, 0, 1, 0.96])
    plt.show(block=False)

program = "serv-hand"

mpl.rcParams["axes.formatter.use_mathtext"] = True
mpl.rcParams.update({
    "font.family": "cmr10"  # Use built-in Computer Modern
})

plot_mem(program)
plot_sys(program)

plt.show()



