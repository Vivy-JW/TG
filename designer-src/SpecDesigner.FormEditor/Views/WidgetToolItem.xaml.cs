using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Markup;

namespace SpecDesigner.FormEditor.Views
{
	// Token: 0x0200002E RID: 46
	public partial class WidgetToolItem : UserControl, IDisposable, ICommandSource
	{
		// Token: 0x0600019B RID: 411 RVA: 0x000089BA File Offset: 0x00006BBA
		public WidgetToolItem()
		{
			this.InitializeComponent();
		}

		// Token: 0x17000055 RID: 85
		// (get) Token: 0x0600019C RID: 412 RVA: 0x000089C8 File Offset: 0x00006BC8
		// (set) Token: 0x0600019D RID: 413 RVA: 0x000089DA File Offset: 0x00006BDA
		public string WidgetType
		{
			get
			{
				return (string)base.GetValue(WidgetToolItem.WidgetTypeProperty);
			}
			set
			{
				base.SetValue(WidgetToolItem.WidgetTypeProperty, value);
			}
		}

		// Token: 0x17000056 RID: 86
		// (get) Token: 0x0600019E RID: 414 RVA: 0x000089E8 File Offset: 0x00006BE8
		// (set) Token: 0x0600019F RID: 415 RVA: 0x000089FA File Offset: 0x00006BFA
		public string ToolImageSource
		{
			get
			{
				return base.GetValue(WidgetToolItem.ToolImageSourceProperty).ToString();
			}
			set
			{
				base.SetValue(WidgetToolItem.ToolImageSourceProperty, value);
			}
		}

		// Token: 0x060001A0 RID: 416 RVA: 0x00008A08 File Offset: 0x00006C08
		public void Dispose()
		{
		}

		// Token: 0x17000057 RID: 87
		// (get) Token: 0x060001A1 RID: 417 RVA: 0x00008A0A File Offset: 0x00006C0A
		// (set) Token: 0x060001A2 RID: 418 RVA: 0x00008A1C File Offset: 0x00006C1C
		public ICommand Command
		{
			get
			{
				return (ICommand)base.GetValue(WidgetToolItem.CommandProperty);
			}
			set
			{
				base.SetValue(WidgetToolItem.CommandProperty, value);
			}
		}

		// Token: 0x17000058 RID: 88
		// (get) Token: 0x060001A3 RID: 419 RVA: 0x00008A2A File Offset: 0x00006C2A
		// (set) Token: 0x060001A4 RID: 420 RVA: 0x00008A37 File Offset: 0x00006C37
		public object CommandParameter
		{
			get
			{
				return base.GetValue(WidgetToolItem.CommandParameterProperty);
			}
			set
			{
				base.SetValue(WidgetToolItem.CommandParameterProperty, value);
			}
		}

		// Token: 0x17000059 RID: 89
		// (get) Token: 0x060001A5 RID: 421 RVA: 0x00008A45 File Offset: 0x00006C45
		// (set) Token: 0x060001A6 RID: 422 RVA: 0x00008A57 File Offset: 0x00006C57
		public IInputElement CommandTarget
		{
			get
			{
				return (IInputElement)base.GetValue(WidgetToolItem.CommandTargetProperty);
			}
			set
			{
				base.SetValue(WidgetToolItem.CommandTargetProperty, value);
			}
		}

		// Token: 0x040000EA RID: 234
		public static readonly DependencyProperty WidgetTypeProperty = DependencyProperty.Register("WidgetType", typeof(string), typeof(WidgetToolItem), null);

		// Token: 0x040000EB RID: 235
		public static readonly DependencyProperty ToolImageSourceProperty = DependencyProperty.Register("ToolImageSource", typeof(string), typeof(WidgetToolItem), null);

		// Token: 0x040000EC RID: 236
		public static readonly DependencyProperty CommandProperty = DependencyProperty.Register("Command", typeof(ICommand), typeof(WidgetToolItem), new UIPropertyMetadata(null));

		// Token: 0x040000ED RID: 237
		public static readonly DependencyProperty CommandParameterProperty = DependencyProperty.Register("CommandParameter", typeof(object), typeof(WidgetToolItem), new UIPropertyMetadata(null));

		// Token: 0x040000EE RID: 238
		public static readonly DependencyProperty CommandTargetProperty = DependencyProperty.Register("CommandTarget", typeof(IInputElement), typeof(WidgetToolItem), new UIPropertyMetadata(null));
	}
}
