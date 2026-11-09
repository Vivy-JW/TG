using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Documents;
using System.Windows.Markup;

namespace SpecDesigner.SpecEditor.Views
{
	// Token: 0x02000013 RID: 19
	public partial class PreviewSpecWindow : Window
	{
		// Token: 0x06000078 RID: 120 RVA: 0x00006737 File Offset: 0x00004937
		public PreviewSpecWindow()
		{
			this.InitializeComponent();
		}

		// Token: 0x1700000C RID: 12
		// (set) Token: 0x06000079 RID: 121 RVA: 0x00006745 File Offset: 0x00004945
		public string StandardSpec
		{
			set
			{
				this.standardSpec.Inlines.Add(value);
			}
		}

		// Token: 0x1700000D RID: 13
		// (set) Token: 0x0600007A RID: 122 RVA: 0x00006758 File Offset: 0x00004958
		public string CiteSpec
		{
			set
			{
				this.citeSpec.Inlines.Add(value);
			}
		}
	}
}
