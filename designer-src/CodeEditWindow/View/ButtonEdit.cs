using System;
using System.Windows;
using System.Windows.Controls;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000009 RID: 9
	public class ButtonEdit : TextBox
	{
		// Token: 0x14000001 RID: 1
		// (add) Token: 0x0600004A RID: 74 RVA: 0x00004B5E File Offset: 0x00002D5E
		// (remove) Token: 0x0600004B RID: 75 RVA: 0x00004B6C File Offset: 0x00002D6C
		public event RoutedEventHandler ButtonClick
		{
			add
			{
				base.AddHandler(ButtonEdit.ButtonClickEvent, value);
			}
			remove
			{
				base.RemoveHandler(ButtonEdit.ButtonClickEvent, value);
			}
		}

		// Token: 0x0600004C RID: 76 RVA: 0x00004B7A File Offset: 0x00002D7A
		public ButtonEdit()
		{
			base.DefaultStyleKey = typeof(ButtonEdit);
		}

		// Token: 0x0600004D RID: 77 RVA: 0x00004B94 File Offset: 0x00002D94
		public override void OnApplyTemplate()
		{
			base.OnApplyTemplate();
			Button button = base.GetTemplateChild("button") as Button;
			if (button != null)
			{
				button.Click += this.button_Click;
			}
		}

		// Token: 0x0600004E RID: 78 RVA: 0x00004BD0 File Offset: 0x00002DD0
		private void button_Click(object sender, RoutedEventArgs e)
		{
			RoutedEventArgs e2 = new RoutedEventArgs(ButtonEdit.ButtonClickEvent);
			base.RaiseEvent(e2);
		}

		// Token: 0x0600004F RID: 79 RVA: 0x00004BEF File Offset: 0x00002DEF
		// Note: this type is marked as 'beforefieldinit'.
		static ButtonEdit()
		{
			ButtonEdit.ButtonClickEvent = EventManager.RegisterRoutedEvent("ButtonClick", RoutingStrategy.Bubble, typeof(RoutedEventHandler), typeof(ButtonEdit));
		}
	}
}
