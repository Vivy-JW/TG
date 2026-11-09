using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000042 RID: 66
	public partial class StandardVersionComparisonWindow : Window, IComparisonWindow
	{
		// Token: 0x17000069 RID: 105
		// (get) Token: 0x060002DC RID: 732 RVA: 0x00018BD1 File Offset: 0x00016DD1
		public static StandardVersionComparisonWindow This
		{
			get
			{
				if (StandardVersionComparisonWindow._this == null)
				{
					StandardVersionComparisonWindow._this = new StandardVersionComparisonWindow();
				}
				return StandardVersionComparisonWindow._this;
			}
		}

		// Token: 0x1700006A RID: 106
		// (get) Token: 0x060002DD RID: 733 RVA: 0x00018BE9 File Offset: 0x00016DE9
		public DiffTextViewer SourceViewer
		{
			get
			{
				return this.textEditor1;
			}
		}

		// Token: 0x1700006B RID: 107
		// (get) Token: 0x060002DE RID: 734 RVA: 0x00018BF1 File Offset: 0x00016DF1
		public DiffTextViewer TargetViewer
		{
			get
			{
				return this.textEditor2;
			}
		}

		// Token: 0x060002DF RID: 735 RVA: 0x00018BF9 File Offset: 0x00016DF9
		public StandardVersionComparisonWindow()
		{
			this.InitializeComponent();
			Application.Current.Exit += this.Current_Exit;
		}

		// Token: 0x060002E0 RID: 736 RVA: 0x00018C1D File Offset: 0x00016E1D
		private new void Show()
		{
			base.Show();
			if (base.Visibility != Visibility.Visible)
			{
				base.Visibility = Visibility.Visible;
			}
			base.Focus();
		}

		// Token: 0x060002E1 RID: 737 RVA: 0x00018C3C File Offset: 0x00016E3C
		public void Show(PackageKey key, ADPModel model)
		{
			base.Title = string.Format("{0} [{1}]", Application.Current.FindResource("CE_DiffOldAndNewStandardContent") as string, key.Program);
			this._diffManager = new DiffManager(this, model.Element.Element("old").Value, model.Element.Element("new").Value);
			this.textEditor1_Title.Content = string.Format("【{0}】Version: {1}", Application.Current.FindResource("CE_BeforePath") as string, model.Element.Element("old").Attribute("std_ver").Value);
			this.textEditor2_Title.Content = string.Format("【{0}】Version: {1}", Application.Current.FindResource("CE_AfterPath") as string, model.Element.Element("new").Attribute("std_ver").Value);
			this.Show();
		}

		// Token: 0x060002E2 RID: 738 RVA: 0x00018D60 File Offset: 0x00016F60
		public void Show(string source, string target)
		{
			base.Title = Application.Current.FindResource("CE_DiffSingleContent") as string;
			this.textEditor1_Title.Content = Application.Current.FindResource("CE_DiffSource") as string;
			this.textEditor2_Title.Content = Application.Current.FindResource("CE_DiffTarget") as string;
			this._diffManager = new DiffManager(this, source, target);
			this.Show();
		}

		// Token: 0x060002E3 RID: 739 RVA: 0x00018DD9 File Offset: 0x00016FD9
		private void Current_Exit(object sender, ExitEventArgs e)
		{
			Application.Current.Exit -= this.Current_Exit;
			base.Close();
		}

		// Token: 0x060002E4 RID: 740 RVA: 0x00018DF7 File Offset: 0x00016FF7
		protected override void OnClosing(CancelEventArgs e)
		{
			e.Cancel = true;
			base.Visibility = Visibility.Hidden;
		}

		// Token: 0x0400013E RID: 318
		private static StandardVersionComparisonWindow _this;

		// Token: 0x0400013F RID: 319
		private DiffManager _diffManager;
	}
}
