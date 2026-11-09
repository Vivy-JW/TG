using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using SpecDesigner.Infrastructure;
using SpecDesignerCommon;

namespace SpecDesigner
{
	// Token: 0x02000016 RID: 22
	public partial class StartArgSetting : Window
	{
		// Token: 0x060001E5 RID: 485 RVA: 0x00008247 File Offset: 0x00006447
		private StartArgSetting()
		{
			this.InitializeComponent();
			Validation.AddErrorHandler(this.startArgBox, new EventHandler<ValidationErrorEventArgs>(this.OnErrorEvent));
		}

		// Token: 0x060001E6 RID: 486 RVA: 0x0000826C File Offset: 0x0000646C
		public StartArgSetting(PackageKey key)
			: this()
		{
			this._programKey = key;
			if (null == this._programKey)
			{
				return;
			}
			base.Title = string.Format("{0} [{1}]", Application.Current.FindResource("Menu_SetCodeStartArg") as string, this._programKey.Program);
			this._model = new DummyArgModel
			{
				StartArg = ResourceController.GetInstance().GetProgramInfo(this._programKey).GetCodeStartArg()
			};
			base.DataContext = this._model;
		}

		// Token: 0x060001E7 RID: 487 RVA: 0x000082F8 File Offset: 0x000064F8
		private void OnErrorEvent(object sender, RoutedEventArgs e)
		{
			ValidationErrorEventArgs e2 = e as ValidationErrorEventArgs;
			switch (e2.Action)
			{
			case ValidationErrorEventAction.Added:
				this._errorCount++;
				break;
			case ValidationErrorEventAction.Removed:
				this._errorCount--;
				break;
			}
			this.acceptButton.IsEnabled = this._errorCount <= 0;
		}

		// Token: 0x060001E8 RID: 488 RVA: 0x00008359 File Offset: 0x00006559
		private void acceptButton_Click(object sender, RoutedEventArgs e)
		{
			ResourceController.GetInstance().GetProgramInfo(this._programKey).SetCodeStartArg(this._model.StartArg);
			base.Close();
		}

		// Token: 0x060001E9 RID: 489 RVA: 0x00008381 File Offset: 0x00006581
		private void cancelButton_Click(object sender, RoutedEventArgs e)
		{
			base.Close();
		}

		// Token: 0x040000B7 RID: 183
		private PackageKey _programKey;

		// Token: 0x040000B8 RID: 184
		private DummyArgModel _model;

		// Token: 0x040000B9 RID: 185
		private int _errorCount;
	}
}
