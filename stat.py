import pandas as pd
from scipy.stats import ttest_rel
import sys

program = sys.argv[1]

goroutines = [1, 16, 32, 64, 128, 256]
res = {
    "G": [],
    "T_C": [],
    "T_L": [],
    "Theta": [],
    "T_A": [],
    "T_D": [],
    "T_C_p" : [],
    "T_L_p" : [],
    "Theta_p" : [],
    "T_A_p": [],
    "T_D_p": []
}
for idx, g in enumerate(goroutines):
    gc_data = pd.read_csv("results/" + program + "/" + str(g) + "-GC-sys.csv")
    rbmm_data = pd.read_csv("results/" + program + "/" + str(g) + "-RBMM-sys.csv")
    metrics = ["T_C", "T_L", "Theta", "T_A", "T_D"]
    print(f"\nPaired t-tests {g} goroutine(s) (GC vs RBMM):\n")
    res["G"].append(g)

    for metric in metrics:
        if metric not in gc_data.columns or metric not in rbmm_data.columns:
            print(f"Skipping {metric}: not found in both files.")
            continue
        gc_values = gc_data[metric]
        rbmm_values = rbmm_data[metric]

        t_stat, p_value = ttest_rel(gc_values, rbmm_values)
        res[metric].append(t_stat)
        res[metric + "_p"].append(p_value)
        print(f"{metric}: t = {t_stat:.4f}, p = {p_value:.4g}")

df = pd.DataFrame(res)
df.to_csv("results/"+program+"/stat.csv")