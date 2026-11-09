using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Linq;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Markup;
using SpecDesignerCommon;

namespace SpecDesigner.FormEditor.ActionDefaults
{
	// Token: 0x0200002D RID: 45
	public partial class ActionDefaultGroup : UserControl, IDisposable
	{
		// Token: 0x17000053 RID: 83
		// (get) Token: 0x06000191 RID: 401 RVA: 0x00008812 File Offset: 0x00006A12
		// (set) Token: 0x06000192 RID: 402 RVA: 0x00008833 File Offset: 0x00006A33
		public PackageKey ProgramKey
		{
			get
			{
				if (this._programKey == null)
				{
					throw new NullReferenceException("程式名稱為NULL!");
				}
				return this._programKey;
			}
			set
			{
				this._programKey = value;
			}
		}

		// Token: 0x17000054 RID: 84
		// (get) Token: 0x06000193 RID: 403 RVA: 0x0000883C File Offset: 0x00006A3C
		// (set) Token: 0x06000194 RID: 404 RVA: 0x00008844 File Offset: 0x00006A44
		public string ActionType
		{
			get
			{
				return this._actionType;
			}
			set
			{
				string text = value;
				if (text == null)
				{
					text = "";
				}
				this._actionType = text;
				string text2 = Application.Current.Resources["ads_" + ((text == "all") ? "all" : text.Substring(0, 2))] as string;
				this.expander.Header = text2;
			}
		}

		// Token: 0x06000195 RID: 405 RVA: 0x000088AA File Offset: 0x00006AAA
		public ActionDefaultGroup()
		{
			this.InitializeComponent();
		}

		// Token: 0x06000196 RID: 406 RVA: 0x000088B8 File Offset: 0x00006AB8
		public void AddAction(ActionDefault act)
		{
			this.container.Children.Add(act);
		}

		// Token: 0x06000197 RID: 407 RVA: 0x000088CC File Offset: 0x00006ACC
		public void ClearActions()
		{
			foreach (ActionDefault actionDefault in this.container.Children.OfType<ActionDefault>())
			{
				actionDefault.Dispose();
			}
			this.container.Children.Clear();
		}

		// Token: 0x06000198 RID: 408 RVA: 0x00008934 File Offset: 0x00006B34
		public void Dispose()
		{
			base.CommandBindings.Clear();
			this.ClearActions();
		}

		// Token: 0x040000E5 RID: 229
		private PackageKey _programKey;

		// Token: 0x040000E6 RID: 230
		private string _actionType;
	}
}
