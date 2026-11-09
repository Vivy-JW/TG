using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using SpecDesigner.FormEditor.ActionDefaults;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x02000002 RID: 2
	public partial class WorkSpace : UserControl, IDisposable
	{
		// Token: 0x06000001 RID: 1 RVA: 0x00002050 File Offset: 0x00000250
		public WorkSpace()
		{
			this.InitializeComponent();
			DesignerProperties.GetIsInDesignMode(this);
		}

		// Token: 0x06000002 RID: 2 RVA: 0x00002065 File Offset: 0x00000265
		public void Dispose()
		{
			this.toolbar.Dispose();
			this.actionDefaults.Dispose();
			this.managedForm.Dispose();
		}
	}
}
