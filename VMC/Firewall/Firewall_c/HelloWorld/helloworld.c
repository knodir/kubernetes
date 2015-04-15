#include <linux/module.h>
#include <linux/kernel.h>

MODULE_LICENSE("GPL");
MODULE_DESCRIPTION("Firewall");
MODULE_AUTHOR("Feifan Chen/quantumlight");
 
int init_module() {
	printk(KERN_INFO "initialize kernel module\n");
	printk(KERN_INFO "hello world!\n");
	return 0;
}
 
void cleanup_module() {
	printk(KERN_INFO "kernel module unloaded.\n");
}
