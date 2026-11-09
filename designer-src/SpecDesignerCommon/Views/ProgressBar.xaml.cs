using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Markup;
using SpecDesignerCommon.ViewModel;

namespace SpecDesignerCommon.Views
{
	// Token: 0x02000012 RID: 18
	public partial class ProgressBar : Window
	{
		// Token: 0x17000013 RID: 19
		// (get) Token: 0x06000078 RID: 120 RVA: 0x00003D35 File Offset: 0x00001F35
		// (set) Token: 0x06000079 RID: 121 RVA: 0x00003D4D File Offset: 0x00001F4D
		public static ProgressBar Instance
		{
			get
			{
				if (ProgressBar._Instance == null)
				{
					ProgressBar._Instance = new ProgressBar();
				}
				return ProgressBar._Instance;
			}
			private set
			{
				ProgressBar._Instance = value;
			}
		}

		// Token: 0x0600007A RID: 122 RVA: 0x00003D55 File Offset: 0x00001F55
		public ProgressBar()
		{
			this.InitializeComponent();
			base.DataContext = ProgressBarViewModel.Instance;
		}

		// Token: 0x04000033 RID: 51
		private static ProgressBar _Instance;
	}
}
